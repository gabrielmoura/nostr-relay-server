--- Nostr Event System Database Schema to implements NIP-40 https://github.com/nostr-protocol/nips/blob/master/40.md
--- This file contains the SQL commands to create the necessary tables and functions for the Nostr event system.
CREATE OR REPLACE FUNCTION nostr_expiration_submission() RETURNS trigger AS
$$
DECLARE
    expiration_tag       JSONB;
    expiration_value     TEXT;
    expiration_timestamp INTEGER;
    unix_now              INTEGER;
BEGIN
    -- Search for the "expiration" tag in the event tags
    SELECT value
    INTO expiration_tag
    FROM jsonb_array_elements(NEW.tags) AS value
    WHERE value ->> 0 = 'expiration'
    LIMIT 1;

    -- If there is no "expiration" tag, it just returns
    IF expiration_tag IS NULL THEN
        RETURN NEW;
    END IF;

    -- Extracts the timestamp (second element of the tag)
    expiration_value := expiration_tag ->> 1;

    -- Try to convert to integer
    BEGIN
        expiration_timestamp := expiration_value::INTEGER;
    EXCEPTION
        WHEN others THEN
            RAISE EXCEPTION 'Invalid expiration timestamp: %', expiration_value;
    END;

    -- Gets the current time in seconds
    SELECT extract(epoch FROM now())::INTEGER INTO unix_now;

    -- Check if it has already expired
    IF expiration_timestamp < unix_now THEN
        RAISE EXCEPTION 'expired event';
    END IF;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

--- Trigger to check if the event expiration is in the past
CREATE OR REPLACE TRIGGER nostr_expiration_submission_trigger
    BEFORE INSERT
    ON event
    FOR EACH ROW
EXECUTE FUNCTION nostr_expiration_submission();