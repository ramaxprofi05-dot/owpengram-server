DROP TRIGGER IF EXISTS user_profile_video_banners_media_reference
    ON public.user_profile_video_banners;

DROP FUNCTION IF EXISTS public.sync_user_profile_video_banner_media_reference();

DROP TRIGGER IF EXISTS user_profile_video_banners_updated_at
    ON public.user_profile_video_banners;

DROP FUNCTION IF EXISTS public.touch_user_profile_video_banner_updated_at();

DROP TABLE IF EXISTS public.user_profile_video_banners;
