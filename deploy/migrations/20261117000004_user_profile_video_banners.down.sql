DROP TRIGGER IF EXISTS user_profile_video_banners_media_reference
    ON public.user_profile_video_banners;

DROP FUNCTION IF EXISTS public.sync_user_profile_video_banner_media_reference();

DROP TABLE IF EXISTS public.user_profile_video_banners;
