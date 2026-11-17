-- A profile banner must reference media uploaded by the same account.
-- Enforce this at the database boundary as well as in the application service,
-- so a future RPC path cannot attach another user's document by mistake.
CREATE FUNCTION public.enforce_user_profile_video_banner_document_owner()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM public.documents
        WHERE id = NEW.document_id
          AND owner_user_id = NEW.owner_user_id
    ) THEN
        RAISE EXCEPTION 'profile video banner document must belong to profile owner'
            USING ERRCODE = '23514';
    END IF;

    RETURN NEW;
END;
$$;

CREATE TRIGGER user_profile_video_banners_document_owner
BEFORE INSERT OR UPDATE OF owner_user_id, document_id
ON public.user_profile_video_banners
FOR EACH ROW
EXECUTE FUNCTION public.enforce_user_profile_video_banner_document_owner();
