-- +migrate Up

CREATE TABLE signers_audit (
  audit_id BIGINT NOT NULL PRIMARY KEY GENERATED ALWAYS AS IDENTITY,
  audit_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
  audit_user TEXT NOT NULL DEFAULT USER,
  audit_op audit_op NOT NULL,
  LIKE signers
);

-- +migrate StatementBegin
CREATE FUNCTION record_signers_audit() RETURNS TRIGGER AS $BODY$
  BEGIN
    IF (TG_OP = 'INSERT') THEN
      INSERT INTO signers_audit VALUES (DEFAULT, DEFAULT, DEFAULT, TG_OP::audit_op, NEW.*);
      RETURN NEW;
    ELSIF (TG_OP = 'UPDATE') THEN
      INSERT INTO signers_audit VALUES (DEFAULT, DEFAULT, DEFAULT, TG_OP::audit_op, NEW.*);
      RETURN NEW;
    ELSIF (TG_OP = 'DELETE') THEN
      INSERT INTO signers_audit VALUES (DEFAULT, DEFAULT, DEFAULT, TG_OP::audit_op, OLD.*);
      RETURN OLD;
    END IF;
  END;
$BODY$ LANGUAGE plpgsql;
-- +migrate StatementEnd

CREATE TRIGGER record_signers_audit
AFTER INSERT OR UPDATE OR DELETE ON signers
  FOR EACH ROW EXECUTE PROCEDURE record_signers_audit();

-- +migrate Down

DROP TRIGGER record_signers_audit ON signers;
DROP FUNCTION record_signers_audit;
DROP TABLE signers_audit;
