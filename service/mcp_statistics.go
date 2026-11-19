package service

import "strings"

// mcpStatisticsField 仅开放各领域已核对的统计及范围字段，错误原文和配置不在其中。
func mcpStatisticsField(tool, field string) bool {
	fields := ""
	switch {
	case strings.HasPrefix(tool, "newapi_channel_analytics_") || tool == "newapi_channel_stability" || tool == "newapi_channel_status_codes" || tool == "newapi_channel_failures":
		fields = `meta generated_at reliable_from_ts data_start_ts data_end_ts last_flushed_at runtime_pending_batch_count runtime_flush_failure_count runtime_last_flush_error_at bucket_level bucket_seconds retention_days partial detail_available uncovered_channel_types invalid_sample_count dimension_overflow_count dropped_metric_event_count dropped_failure_event_count dimension_hash_collision_count backfill backfill_start_ts live_cutover_ts total_rows scanned_rows converted_rows skipped_rows metric_bucket_count failure_event_count completed_at final_request_count channel_attempt_count upstream_call_count failed_attempt_count client_success_rate channel_quality_success_rate attempt_success_rate retry_rate usage_sample_count input_tokens_total uncached_input_tokens cache_read_tokens cache_write_tokens cache_request_hit_rate cache_token_hit_rate charged_quota charged_micro_usd avg_latency_ms p95_latency_ms avg_ttft_ms p95_ttft_ms bucket_ts event_count status_present label error_stages error_stage channel_type channel_type_name failure_count last_failure_at top_status_codes requested_model upstream_model windows window_seconds start_timestamp end_timestamp quality_eligible_count quality_success_count quality_success_rate attempt_eligible_count attempt_success_count partial_response_count upstream_status_sample_count upstream_status_coverage_rate upstream_429_count upstream_4xx_count upstream_5xx_count http_error_count transport_error_count protocol_error_count stream_error_count local_error_count dispatch_error_count client_cancelled_count live_event_count legacy_event_count live_event_rate legacy_event_rate minimum_sample_count sample_sufficient latency_sample_count latency_coverage_rate ttft_sample_count ttft_coverage_rate last_failure_bucket_ts dimension event_id attempt_seq retry_planned is_last_started_attempt traffic_source data_origin outcome failure_owner quality_eligible partial_response upstream_status_present upstream_status_code normalized_status_present normalized_status_code client_status_present client_status_code ttft_present ttft_ms`
	case tool == "newapi_revenue":
		fields = "data_points online_money redemption_quota online_count redemption_count total_online_money total_redemption_quota total_online_count total_redemption_count"
	case strings.HasPrefix(tool, "newapi_notification_"):
		fields = "event_type event_name task_name bot_id bot_name event_id target_id attempt_count next_attempt_at sent_at last_triggered_at source_id"
	case tool == "newapi_system_instances":
		fields = "last_seen_at stale_after_seconds"
	case tool == "newapi_active_requests":
		fields = "observed_at total_active total_requests active_count instance_count request_id started_at elapsed_ms stage requested_model"
	case tool == "newapi_status":
		fields = "system_name start_time enable_task enable_drawing setup"
	}
	return fields != "" && strings.Contains(" "+fields+" ", " "+field+" ")
}
