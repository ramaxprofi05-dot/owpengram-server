-- One public, user-selected video banner per profile. The media itself stays
-- in the existing documents/object-storage pipeline; this table only owns the
-- profile-to-document relationship and lifecycle metadata.
CREATE TABLE public.user_profile_video_banners (
    owner_user_id bigint PRIMARY KEY REFERENCES public.users(id) ON DELETE CASCADE,
    document_id bigint NOT NULL UNIQUE REFERENCES public.documents(id) ON DELETE RESTRICT,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT user_profile_video_banners_owner_check CHECK (owner_user_id > 0)
);

CREATE INDEX user_profile_video_banners_document_idx
    ON public.user_profile_video_banners(document_id);

-- Keep the existing media-retention system aware of the banner reference on
-- every insert, replacement, and removal. This is transactional with the row.
CREATE FUNCTION public.sync_user_profile_video_banner_media_reference()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
    IF TG_OP = 'DELETE' THEN
        DELETE FROM public.media_references
        WHERE media_kind = 'document'
          AND media_id = OLD.document_id
          AND ref_kind = 'profile_video_banner'
          AND ref_key = OLD.owner_user_id::text;
        RETURN OLD;
    END IF;

    IF TG_OP = 'UPDATE' AND OLD.document_id IS DISTINCT FROM NEW.document_id THEN
        DELETE FROM public.media_references
        WHERE media_kind = 'document'
          AND media_id = OLD.document_id
          AND ref_kind = 'profile_video_banner'
          AND ref_key = OLD.owner_user_id::text;
    END IF;

    INSERT INTO public.media_references(media_kind, media_id, ref_kind, ref_key)
    VALUES ('document', NEW.document_id, 'profile_video_banner', NEW.owner_user_id::text)
    ON CONFLICT DO NOTHING;
    RETURN NEW;
END;
$$;

CREATE TRIGGER user_profile_video_banners_media_reference
AFTER INSERT OR UPDATE OR DELETE ON public.user_profile_video_banners
FOR EACH ROW EXECUTE FUNCTION public.sync_user_profile_video_banner_media_reference();
