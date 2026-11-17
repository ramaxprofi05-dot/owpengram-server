DROP TRIGGER IF EXISTS user_profile_video_banners_document_owner
    ON public.user_profile_video_banners;

DROP FUNCTION IF EXISTS public.enforce_user_profile_video_banner_document_owner();
