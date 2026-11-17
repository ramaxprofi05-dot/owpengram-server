-- Rebrand the built-in system account for FemboyGram.
-- Fail clearly rather than stealing a username if an existing account already
-- owns the reserved product handle in a database imported from another node.
DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM public.users
        WHERE lower(username) = 'femboygram' AND id <> 777000
    ) THEN
        RAISE EXCEPTION 'cannot assign @femboygram to system account 777000: username is already occupied';
    END IF;
END
$$;

UPDATE public.users
SET first_name = 'FemboyGram', username = 'femboygram', updated_at = now()
WHERE id = 777000;
