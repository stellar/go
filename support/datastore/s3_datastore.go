package datastore

import (
	"bytes"
	"context"
	"crypto/tls"
	"errors"
	"io"
	stLog "log" // Import Go standard log
	"net/http"
	"os"
	"path"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/smithy-go"
	"github.com/aws/smithy-go/logging"
	stellarErrors "github.com/stellar/go/support/errors"
	"github.com/stellar/go/support/log"
)

// S3DataStore implements the DataStore interface for AWS S3.
// It provides methods to interact with S3 buckets for storing and retrieving ledger data.
type S3DataStore struct {
	s3Uploader *manager.Uploader // Use manager for multipart uploads
	s3Client   *s3.Client
	bucket     string
	schema     DataStoreSchema
	// Optional parameters from config
	prefix       string // prefix for all objects in the bucket
	sse          string // ServerSideEncryption (e.g., "AES256", "aws:kms")
	sseKmsKeyID  string // Required if sse is "aws:kms"
	storageClass string // S3 Storage Class (e.g., "STANDARD", "INTELLIGENT_TIERING")
	acl          string // S3 Object Canned ACL (e.g., "private", "public-read")
}

// Compile-time check to ensure S3DataStore implements the DataStore interface.
var _ DataStore = (*S3DataStore)(nil)

// NewS3DataStore creates a new instance of S3DataStore.
// It initializes the S3 client and uploader based on the provided parameters.
//
// ctx is the context for the operation.
// params is a map of configuration parameters for S3, including:
//   - "destination_bucket": (Required) The name of the S3 bucket.
//   - "region": (Required) The AWS region of the bucket.
//   - "aws_profile": (Optional) The AWS shared config profile to use.
//   - "endpoint_url": (Optional) A custom S3 endpoint URL (for S3-compatible storage or local testing).
//   - "disable_ssl": (Optional) "true" to disable SSL for custom endpoints (use with caution).
//   - "force_path_style": (Optional) "true" to force path-style addressing for S3.
//   - "destination_path_prefix": (Optional) A prefix for all object keys within the bucket.
//   - "server_side_encryption": (Optional) S3 server-side encryption type (e.g., "AES256").
//   - "sse_kms_key_id": (Optional) KMS Key ID if server_side_encryption is "aws:kms".
//   - "storage_class": (Optional) S3 storage class for uploaded objects.
//   - "acl": (Optional) Canned ACL for uploaded objects.
//
// schema defines the structure for storing data, like ledgers per file.
func NewS3DataStore(ctx context.Context, params map[string]string, schema DataStoreSchema) (*S3DataStore, error) {
	bucket, ok := params["destination_bucket"]
	if !ok {
		return nil, stellarErrors.New("S3 config missing required parameter: destination_bucket")
	}
	region, ok := params["region"]
	if !ok {
		return nil, stellarErrors.New("S3 config missing required parameter: region")
	}

	awsProfile := params["aws_profile"]                              // Optional
	endpointURL := params["endpoint_url"]                            // Read endpoint_url
	disableSSLStr := strings.ToLower(params["disable_ssl"])          // Read disable_ssl (as string)
	forcePathStyleStr := strings.ToLower(params["force_path_style"]) // Read force_path_style (as string)

	var disableSSL bool
	if disableSSLStr == "true" {
		disableSSL = true
	}

	var forcePathStyle bool
	if forcePathStyleStr == "true" {
		forcePathStyle = true
	}

	// Prepare config loaders
	configLoaders := []func(*config.LoadOptions) error{
		config.WithRegion(region),
		config.WithSharedConfigProfile(awsProfile),
		// Configure AWS SDK logger to use standard Go logger (output to stderr)
		config.WithLogger(logging.NewStandardLogger(stLog.Writer())),
	}

	// --- Handle Custom Endpoint ---
	if endpointURL != "" {
		log.Infof("Using custom S3 endpoint: %s", endpointURL)
		// Custom resolver function
		endpointResolver := aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
			if service == s3.ServiceID {
				return aws.Endpoint{
					URL:               endpointURL,
					HostnameImmutable: true, // Important for preventing bucket name prepending
					Source:            aws.EndpointSourceCustom,
				}, nil
			}
			// fallback to default resolution
			return aws.Endpoint{}, &aws.EndpointNotFoundError{}
		})
		configLoaders = append(configLoaders, config.WithEndpointResolverWithOptions(endpointResolver))
	}

	// --- Handle Disable SSL ---
	if disableSSL {
		log.Info("Disabling SSL verification for S3 endpoint (intended for local testing)")
		// Note: This requires providing a custom HTTP client. Be cautious using this in production.
		// We'll create a default client and modify its transport.
		customHTTPClient := &http.Client{}
		// transport can be nil if not explicitly set.
		// http.DefaultTransport is an *http.Transport.
		baseTransport := http.DefaultTransport.(*http.Transport)
		if baseTransport == nil {
			// This case should ideally not happen with http.DefaultTransport
			baseTransport = &http.Transport{}
		}
		clonedTransport := baseTransport.Clone()
		if clonedTransport.TLSClientConfig == nil {
			clonedTransport.TLSClientConfig = &tls.Config{}
		}
		clonedTransport.TLSClientConfig.InsecureSkipVerify = true // #nosec G402 - Intentionally skipping verification for local dev
		customHTTPClient.Transport = clonedTransport
		configLoaders = append(configLoaders, config.WithHTTPClient(customHTTPClient))
	}

	// Load the final config
	cfg, err := config.LoadDefaultConfig(ctx, configLoaders...)
	if err != nil {
		return nil, stellarErrors.Wrap(err, "failed to load AWS config")
	}

	// --- Handle Path Style Addressing ---
	s3ClientOptions := []func(*s3.Options){
		func(o *s3.Options) {
			// Check both the config param AND the standard env var AWS_S3_USE_PATH_STYLE
			// Environment variable takes precedence if set.
			usePathStyleEnv := os.Getenv("AWS_S3_USE_PATH_STYLE")
			if usePathStyleEnv == "true" || (usePathStyleEnv == "" && forcePathStyle) {
				log.Info("Using Path-Style S3 addressing")
				o.UsePathStyle = true
			}
		},
	}

	s3Client := s3.NewFromConfig(cfg, s3ClientOptions...)
	s3Uploader := manager.NewUploader(s3Client)

	store := &S3DataStore{
		s3Uploader:   s3Uploader,
		s3Client:     s3Client,
		bucket:       bucket,
		schema:       schema,
		prefix:       params["destination_path_prefix"],
		sse:          params["server_side_encryption"],
		sseKmsKeyID:  params["sse_kms_key_id"],
		storageClass: params["storage_class"],
		acl:          params["acl"],
	}

	return store, nil
}

// fullPath joins the prefix with the object path
func (s *S3DataStore) fullPath(objectPath string) string {
	// Use path.Join for cleaning, but ensure no leading slash if prefix is empty
	// and ensure objectPath is treated as relative to the prefix.
	trimmedObjectPath := strings.TrimPrefix(objectPath, "/")
	if s.prefix == "" {
		return trimmedObjectPath
	}
	// path.Join cleans multiple slashes and relative paths.
	// If objectPath starts with "/", path.Join might treat it as absolute,
	// so ensure objectPath is relative if prefix is used.
	return path.Join(s.prefix, trimmedObjectPath)
}

// GetFileMetadata retrieves the custom user metadata associated with an S3 object.
func (s *S3DataStore) GetFileMetadata(ctx context.Context, objectPath string) (map[string]string, error) {
	fullPath := s.fullPath(objectPath)
	headInput := &s3.HeadObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(fullPath),
	}

	headOutput, err := s.s3Client.HeadObject(ctx, headInput)
	if err != nil {
		var apiErr smithy.APIError
		if errors.As(err, &apiErr) && (apiErr.ErrorCode() == "NotFound" || apiErr.ErrorCode() == "NoSuchKey") {
			return nil, stellarErrors.Wrapf(os.ErrNotExist, "object not found at path: %s", fullPath)
		}
		return nil, stellarErrors.Wrapf(err, "failed to head S3 object: %s", fullPath)
	}

	return headOutput.Metadata, nil
}

// GetFile retrieves an S3 object's content as a readable stream.
func (s *S3DataStore) GetFile(ctx context.Context, objectPath string) (io.ReadCloser, error) {
	fullPath := s.fullPath(objectPath)
	getInput := &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(fullPath),
	}

	getOutput, err := s.s3Client.GetObject(ctx, getInput)
	if err != nil {
		var apiErr smithy.APIError
		if errors.As(err, &apiErr) && (apiErr.ErrorCode() == "NotFound" || apiErr.ErrorCode() == "NoSuchKey") {
			return nil, stellarErrors.Wrapf(os.ErrNotExist, "object not found at path: %s", fullPath)
		}
		return nil, stellarErrors.Wrapf(err, "failed to get S3 object: %s", fullPath)
	}

	return getOutput.Body, nil
}

// PutFile uploads data to S3 using the S3 Upload Manager for automatic multipart handling.
// The 'in' parameter is an io.WriterTo, allowing efficient writing of data.
// metaData is a map of custom metadata to associate with the S3 object.
func (s *S3DataStore) PutFile(ctx context.Context, objectPath string, in io.WriterTo, metaData map[string]string) error {
	fullPath := s.fullPath(objectPath)

	buf := new(bytes.Buffer) // Buffer to hold data from io.WriterTo
	_, err := in.WriteTo(buf)
	if err != nil {
		return stellarErrors.Wrap(err, "failed to write data to buffer for S3 upload")
	}

	uploadInput := &s3.PutObjectInput{
		Bucket:   aws.String(s.bucket),
		Key:      aws.String(fullPath),
		Body:     bytes.NewReader(buf.Bytes()), // Use the buffered data
		Metadata: metaData,
	}

	if s.sse != "" {
		uploadInput.ServerSideEncryption = types.ServerSideEncryption(s.sse)
		if types.ServerSideEncryption(s.sse) == types.ServerSideEncryptionAwsKms && s.sseKmsKeyID != "" {
			uploadInput.SSEKMSKeyId = aws.String(s.sseKmsKeyID)
		}
	}
	if s.storageClass != "" {
		uploadInput.StorageClass = types.StorageClass(s.storageClass)
	}
	if s.acl != "" {
		uploadInput.ACL = types.ObjectCannedACL(s.acl)
	}

	_, err = s.s3Uploader.Upload(ctx, uploadInput)
	if err != nil {
		return stellarErrors.Wrapf(err, "failed to upload object to S3: %s", fullPath)
	}

	return nil
}

// PutFileIfNotExists uploads data only if the object does not already exist in S3.
// It returns true if the file was uploaded, false if it already existed.
func (s *S3DataStore) PutFileIfNotExists(ctx context.Context, objectPath string, in io.WriterTo, metaData map[string]string) (bool, error) {
	exists, err := s.Exists(ctx, objectPath)
	if err != nil {
		// Don't wrap os.ErrNotExist, allow caller to check for it.
		if errors.Is(err, os.ErrNotExist) {
			// If Exists check itself says not found, proceed to upload.
			// This specific path in Exists might not be hit if HeadObject directly returns NotFound.
		} else {
			return false, stellarErrors.Wrapf(err, "failed check for existence before put: %s", objectPath)
		}
	}

	if exists {
		return false, nil // Object already exists, nothing to do.
	}

	// Object does not exist, proceed with upload.
	err = s.PutFile(ctx, objectPath, in, metaData)
	if err != nil {
		return false, err // Return the upload error
	}

	return true, nil
}

// Exists checks if an object exists in S3 using HeadObject.
func (s *S3DataStore) Exists(ctx context.Context, objectPath string) (bool, error) {
	fullPath := s.fullPath(objectPath)
	headInput := &s3.HeadObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(fullPath),
	}

	_, err := s.s3Client.HeadObject(ctx, headInput)
	if err != nil {
		var apiErr smithy.APIError
		if errors.As(err, &apiErr) && (apiErr.ErrorCode() == "NotFound" || apiErr.ErrorCode() == "NoSuchKey") {
			return false, nil // Object does not exist
		}
		// For other errors, wrap and return them.
		return false, stellarErrors.Wrapf(err, "failed to check existence of S3 object: %s", fullPath)
	}

	return true, nil // Object exists
}

// Size returns the size of an object in S3 using HeadObject.
func (s *S3DataStore) Size(ctx context.Context, objectPath string) (int64, error) {
	fullPath := s.fullPath(objectPath)
	headInput := &s3.HeadObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(fullPath),
	}

	headOutput, err := s.s3Client.HeadObject(ctx, headInput)
	if err != nil {
		var apiErr smithy.APIError
		if errors.As(err, &apiErr) && (apiErr.ErrorCode() == "NotFound" || apiErr.ErrorCode() == "NoSuchKey") {
			return 0, stellarErrors.Wrapf(os.ErrNotExist, "object not found at path: %s", fullPath)
		}
		return 0, stellarErrors.Wrapf(err, "failed to get size of S3 object: %s", fullPath)
	}

	if headOutput.ContentLength == nil {
		// AWS SDK V2 uses *int64 for ContentLength, so it can be nil.
		// This case should be rare for existing objects but good to handle.
		return 0, stellarErrors.Errorf("S3 HeadObject returned nil ContentLength for path: %s", fullPath)
	}
	return *headOutput.ContentLength, nil
}

// GetSchema returns the DataStoreSchema associated with this S3DataStore.
func (s *S3DataStore) GetSchema() DataStoreSchema {
	return s.schema
}

// Close performs any necessary cleanup for the S3DataStore.
// For the AWS SDK v2 S3 client, explicit closing is not typically required.
func (s *S3DataStore) Close() error {
	// S3 client from aws-sdk-go-v2 does not require explicit closing.
	// The underlying HTTP client might be shared and managed by the SDK.
	return nil
}
