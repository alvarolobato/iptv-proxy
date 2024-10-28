package xtreamcodes

const (
	StructFields = "Fields"
)

// ServerInfoFields contains constants for ServerInfo struct JSON field names
const (
	ServerInfoFieldHTTPSPort    = "https_port"
	ServerInfoFieldPort         = "port"
	ServerInfoFieldProcess      = "process"
	ServerInfoFieldRTMPPort     = "rtmp_port"
	ServerInfoFieldProtocol     = "server_protocol"
	ServerInfoFieldTimeNow      = "time_now"
	ServerInfoFieldTimestampNow = "timestamp_now"
	ServerInfoFieldTimezone     = "timezone"
	ServerInfoFieldURL          = "url"
)

// UserInfoFields contains constants for UserInfo struct JSON field names
const (
	UserInfoFieldActiveConnections    = "active_cons"
	UserInfoFieldAllowedOutputFormats = "allowed_output_formats"
	UserInfoFieldAuth                 = "auth"
	UserInfoFieldCreatedAt            = "created_at"
	UserInfoFieldExpDate              = "exp_date"
	UserInfoFieldIsTrial              = "is_trial"
	UserInfoFieldMaxConnections       = "max_connections"
	UserInfoFieldMessage              = "message"
	UserInfoFieldPassword             = "password"
	UserInfoFieldStatus               = "status"
	UserInfoFieldUsername             = "username"
)

// CategoryFields contains constants for Category struct JSON field names
const (
	CategoryFieldID     = "category_id"
	CategoryFieldName   = "category_name"
	CategoryFieldParent = "parent_id"
)

// StreamFields contains constants for Stream struct JSON field names
const (
	StreamFieldAdded              = "added"
	StreamFieldCategoryID         = "category_id"
	StreamFieldCategoryName       = "category_name"
	StreamFieldContainerExtension = "container_extension"
	StreamFieldCustomSid          = "custom_sid"
	StreamFieldDirectSource       = "direct_source"
	StreamFieldEPGChannelID       = "epg_channel_id"
	StreamFieldIcon               = "stream_icon"
	StreamFieldID                 = "stream_id"
	StreamFieldName               = "name"
	StreamFieldNumber             = "num"
	StreamFieldRating             = "rating"
	StreamFieldRating5based       = "rating_5based"
	StreamFieldTVArchive          = "tv_archive"
	StreamFieldTVArchiveDuration  = "tv_archive_duration"
	StreamFieldType               = "stream_type"
)

// SeriesInfoFields contains constants for SeriesInfo struct JSON field names
const (
	SeriesInfoFieldBackdropPath   = "backdrop_path"
	SeriesInfoFieldCast           = "cast"
	SeriesInfoFieldCategoryID     = "category_id"
	SeriesInfoFieldCover          = "cover"
	SeriesInfoFieldDirector       = "director"
	SeriesInfoFieldEpisodeRunTime = "episode_run_time"
	SeriesInfoFieldGenre          = "genre"
	SeriesInfoFieldLastModified   = "last_modified"
	SeriesInfoFieldName           = "name"
	SeriesInfoFieldNum            = "num"
	SeriesInfoFieldPlot           = "plot"
	SeriesInfoFieldRating         = "rating"
	SeriesInfoFieldRating5        = "rating_5based"
	SeriesInfoFieldReleaseDate    = "releaseDate"
	SeriesInfoFieldSeriesID       = "series_id"
	SeriesInfoFieldStreamType     = "stream_type"
	SeriesInfoFieldYoutubeTrailer = "youtube_trailer"
)

// SeriesEpisodeFields contains constants for SeriesEpisode struct JSON field names
const (
	SeriesEpisodeFieldAdded              = "added"
	SeriesEpisodeFieldContainerExtension = "container_extension"
	SeriesEpisodeFieldCustomSid          = "custom_sid"
	SeriesEpisodeFieldDirectSource       = "direct_source"
	SeriesEpisodeFieldEpisodeNum         = "episode_num"
	SeriesEpisodeFieldID                 = "id"
	SeriesEpisodeFieldInfo               = "info"
	SeriesEpisodeFieldSeason             = "season"
	SeriesEpisodeFieldTitle              = "title"
)

// SeriesFields contains constants for Series struct JSON field names
const (
	SeriesFieldEpisodes = "episodes"
	SeriesFieldInfo     = "info"
	SeriesFieldSeasons  = "seasons"
)

// VideoOnDemandInfoFields contains constants for VideoOnDemandInfo struct JSON field names
const (
	VideoOnDemandInfoFieldInfo      = "info"
	VideoOnDemandInfoFieldMovieData = "movie_data"
)

// VODInfoFields contains constants for VODInfo struct JSON field names
const (
	VODInfoFieldAudio          = "audio"
	VODInfoFieldBackdropPath   = "backdrop_path"
	VODInfoFieldBitrate        = "bitrate"
	VODInfoFieldCast           = "cast"
	VODInfoFieldDirector       = "director"
	VODInfoFieldDuration       = "duration"
	VODInfoFieldDurationSecs   = "duration_secs"
	VODInfoFieldGenre          = "genre"
	VODInfoFieldMovieImage     = "movie_image"
	VODInfoFieldPlot           = "plot"
	VODInfoFieldRating         = "rating"
	VODInfoFieldReleaseDate    = "releasedate"
	VODInfoFieldTmdbID         = "tmdb_id"
	VODInfoFieldVideo          = "video"
	VODInfoFieldYoutubeTrailer = "youtube_trailer"
)

// EPGInfoFields contains constants for EPGInfo struct JSON field names
const (
	EPGInfoFieldChannelID      = "channel_id"
	EPGInfoFieldDescription    = "description"
	EPGInfoFieldEnd            = "end"
	EPGInfoFieldEPGID          = "epg_id"
	EPGInfoFieldHasArchive     = "has_archive"
	EPGInfoFieldID             = "id"
	EPGInfoFieldLang           = "lang"
	EPGInfoFieldNowPlaying     = "now_playing"
	EPGInfoFieldStart          = "start"
	EPGInfoFieldStartTimestamp = "start_timestamp"
	EPGInfoFieldStopTimestamp  = "stop_timestamp"
	EPGInfoFieldTitle          = "title"
)

// EpisodeInfoFields contains constants for EpisodeInfo struct JSON field names
const (
	EpisodeInfoFieldAudio        = "audio"
	EpisodeInfoFieldBitrate      = "bitrate"
	EpisodeInfoFieldDuration     = "duration"
	EpisodeInfoFieldDurationSecs = "duration_secs"
	EpisodeInfoFieldMovieImage   = "movie_image"
	EpisodeInfoFieldName         = "name"
	EpisodeInfoFieldPlot         = "plot"
	EpisodeInfoFieldRating       = "rating"
	EpisodeInfoFieldReleaseDate  = "releasedate"
	EpisodeInfoFieldVideo        = "video"
)
