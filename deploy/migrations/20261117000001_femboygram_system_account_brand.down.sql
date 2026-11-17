-- Restore the upstream OwpenGram system identity.
DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM public.users
        WHERE lower(username) = 'owpengram' AND id <> 777000
    ) THEN
        RAISE EXCEPTION 'cannot restore @owpengram to system account 777000: username is already occupied';
    END IF;
END
$$;

UPDATE public.users
SET first_name = 'OwpenGram', username = 'owpengram', updated_at = now()
WHERE id = 777000;
