require 'bundler'
Bundler.setup()
require 'pry'

namespace :xdr do

  # As stellar-core adds more .x files, we'll need to update this array
  # Prior to launch, we should be separating our .x files into a separate
  # repo, and should be able to improve this integration.
  HAYASHI_XDR = [
                 "src/xdr/Stellar-types.x",
                 "src/xdr/Stellar-ledger-entries.x",
                 "src/xdr/Stellar-transaction.x",
                 "src/xdr/Stellar-ledger.x",
                 "src/xdr/Stellar-overlay.x",
                 "src/xdr/Stellar-SCP.x",
                ]
  LOCAL_XDR_PATHS = HAYASHI_XDR.map{ |src| "xdr/" + File.basename(src) }

  task :update => [:download, :generate]

  task :download do
    require 'octokit'
    require 'base64'
    FileUtils.mkdir_p "xdr"
    FileUtils.rm_rf "xdr/*.x"

    client = Octokit::Client.new(:netrc => true)

    HAYASHI_XDR.each do |src|
      local_path = "xdr/" + File.basename(src)
      encoded    = client.contents("stellar/stellar-core", path: src).content
      decoded    = Base64.decode64 encoded

      IO.write(local_path, decoded)
    end
  end

  task :generate do
    require "pathname"
    require "xdrgen"
    require 'fileutils'
    FileUtils.rm_f("xdr/xdr_generated.go")

    compilation = Xdrgen::Compilation.new(
      LOCAL_XDR_PATHS,
      output_dir: "xdr",
      namespace:  "xdr",
      language:   :go
    )
    compilation.compile
    system("gofmt -w xdr/xdr_generated.go")
  end
end
