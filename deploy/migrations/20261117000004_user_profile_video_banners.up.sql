-- One public, user-selected video banner per profile. The media itself stays
-- in the existing documents/object-storage pipeline; this table only owns
-- the profile-to-document relationship and lifecycle metadata.
CREATE TABLE public.user_profile_video_banners (
    owner_user_id bigint PRIMARY KEY REFERENCES public.users(id) ON DELETE CASCADE,
    document_id bigint NOT NULL UNIQUE REFERENCES public.documents(id) ON DELETE RESTRICT,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT user_profile_video_banners_owner_check CHECK (owner_user_id > 0)
);

CREATE INDEX user_profile_video_banners_document_idx
    ON public.user_profile_video_banners(document_id);

CREATE FUNCTION public.touch_user_profile_video_banner_updated_at()
RETURNS trigger
LANGUAGE plpgsql
AS $
BEGIN
    NEW.updated_at = now();
    RETURN NEW;
END;
$;

CREATE TRIGGER user_profile_video_banners_updated_at
BEFORE UPDATE ON public.user_profile_video_banners
FOR EACH ROW EXECUTE FUNCTION public.touch_user_profile_video_banner_updated_at();

-- Keep the existing media-retention system aware of the banner reference.
-- Removing/replacing a banner marks the old document orphaned only when no
-- other feature still references it. This mirrors the existing media store
-- lifecycle and prevents both accidental collection and permanent leaks.
CREATE FUNCTION public.sync_user_profile_video_banner_media_reference()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
    IF TG_OP = 'DELETE' OR
       (TG_OP = 'UPDATE' AND OLD.document_id IS DISTINCT FROM NEW.document_id) THEN
        DELETE FROM public.media_references
        WHERE media_kind = 'document'
          AND media_id = OLD.document_id
          AND ref_kind = 'profile_video_banner'
          AND ref_key = OLD.owner_user_id::text;

        UPDATE public.documents
        SET orphaned_at = now()
        WHERE id = OLD.document_id
          AND orphaned_at IS NULL
          AND NOT EXISTS (
              SELECT 1 FROM public.media_references
              WHERE media_kind = 'document' AND media_id = OLD.document_id
          );

        IF TG_OP = 'DELETE' THEN
            RETURN OLD;
        END IF;
    END IF;

    INSERT INTO public.media_references(media_kind, media_id, ref_kind, ref_key)
    VALUES ('document', NEW.document_id, 'profile_video_banner', NEW.owner_user_id::text)
    ON CONFLICT DO NOTHING;

    -- A document may have been marked orphaned before it was selected again.
    UPDATE public.documents
    SET orphaned_at = NULL
    WHERE id = NEW.document_id
      AND orphaned_at IS NOT NULL;

    RETURN NEW;
END;
$$;

CREATE TRIGGER user_profile_video_banners_media_reference
AFTER INSERT OR UPDATE OR DELETE ON public.user_profile_video_banners
FOR EACH ROW EXECUTE FUNCTION public.sync_user_profile_video_banner_media_reference();
