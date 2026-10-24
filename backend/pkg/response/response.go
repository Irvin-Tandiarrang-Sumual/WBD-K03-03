package response

import "github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/pkg/pagination"

type PagingMetadata struct {
	CurrentPage int   `json:"current_page"`
	TotalPage   int   `json:"total_page"`
	Limit       int   `json:"limit"`
	TotalData   int64 `json:"total_data"`
}

type BaseResponse[T any] struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Data    T      `json:"data,omitempty"`
}

type PagedResponse[T any] struct {
	Code   int            `json:"code"`
	Status string         `json:"status"`
	Data   []T            `json:"data"`
	Paging PagingMetadata `json:"paging"`
}

func Success[T any](message string, data T) BaseResponse[T] {
	return BaseResponse[T]{
		Success: true,
		Message: message,
		Data:    data,
	}
}

func Error(message string) BaseResponse[any] {
	return BaseResponse[any]{
		Success: false,
		Message: message,
		Data:    nil,
	}
}

func NewPagedResponse[T any](code int, status string, data []T, params pagination.PaginationParams, totalData int64) PagedResponse[T] {
	totalPage := int(totalData) / params.Limit
	if int(totalData) % params.Limit != 0 {
		totalPage++
	}

	return PagedResponse[T]{
		Code:   code,
		Status: status,
		Data:   data,
		Paging: PagingMetadata{
			CurrentPage: params.Page,
			Limit:       params.Limit,
			TotalPage:   totalPage,
			TotalData:   totalData,
		},
	}
}