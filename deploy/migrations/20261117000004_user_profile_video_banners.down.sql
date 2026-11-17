DELETE FROM public.media_references
WHERE media_kind = 'document' AND ref_kind = 'profile_video_banner';

DROP TABLE IF EXISTS public.user_profile_video_banners;
