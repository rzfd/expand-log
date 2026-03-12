package pagination

import (
	"strconv"

	"github.com/rzfd/expand/internal/pkg/logging"
)

const (
	defaultPage     = 1
	defaultPageSize = 20
	maxPageSize     = 100
)

type Params struct {
	Page     int `json:"page"`
	PageSize int `json:"page_size"`
	Offset   int `json:"-"`
}

type Meta struct {
	Page      int `json:"page"`
	PageSize  int `json:"page_size"`
	Total     int `json:"total"`
	TotalPage int `json:"total_page"`
}

func Parse(pageParam, sizeParam string) Params {
	logging.FromContext(nil).Info().Str("page", pageParam).Str("page_size", sizeParam).Msg("pagination parse started")
	page := parsePositive(pageParam, defaultPage)
	pageSize := parsePositive(sizeParam, defaultPageSize)
	if pageSize > maxPageSize {
		pageSize = maxPageSize
	}

	result := Params{
		Page:     page,
		PageSize: pageSize,
		Offset:   (page - 1) * pageSize,
	}
	logging.FromContext(nil).Info().Int("page", result.Page).Int("page_size", result.PageSize).Int("offset", result.Offset).Msg("pagination parse completed")
	return result
}

func BuildMeta(params Params, total int) Meta {
	logging.FromContext(nil).Info().Int("page", params.Page).Int("page_size", params.PageSize).Int("total", total).Msg("pagination build meta started")
	totalPage := total / params.PageSize
	if total%params.PageSize != 0 {
		totalPage++
	}
	if total == 0 {
		totalPage = 0
	}

	meta := Meta{
		Page:      params.Page,
		PageSize:  params.PageSize,
		Total:     total,
		TotalPage: totalPage,
	}
	logging.FromContext(nil).Info().Int("total_page", meta.TotalPage).Msg("pagination build meta completed")
	return meta
}

func parsePositive(value string, fallback int) int {
	logging.FromContext(nil).Info().Str("value", value).Int("fallback", fallback).Msg("pagination parse positive started")
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed <= 0 {
		logging.FromContext(nil).Info().Str("value", value).Int("result", fallback).Msg("pagination parse positive fallback")
		return fallback
	}
	logging.FromContext(nil).Info().Int("result", parsed).Msg("pagination parse positive completed")
	return parsed
}
