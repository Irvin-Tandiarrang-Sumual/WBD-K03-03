package ginutil

import (
	"strconv"

	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/pkg/pagination"
	"github.com/gin-gonic/gin"
)

func ParseQueryInt(c *gin.Context, key string) (int, error) {
	value := c.Query(key)
	if value == "" {
		return 0, nil
	}
	return strconv.Atoi(value)
}

func ParseUintParam(c *gin.Context, key string) (uint, error) {
	value, err := strconv.ParseUint(c.Param(key), 10, 64)
	if err != nil || value == 0 || uint64(uint(value)) != value {
		return 0, strconv.ErrSyntax
	}
	return uint(value), nil
}

func ParsePagination(c *gin.Context) (pagination.PaginationParams, error) {
	page, err := ParseQueryInt(c, "page")
	if err != nil {
		return pagination.PaginationParams{}, err
	}

	limit, err := ParseQueryInt(c, "limit")
	if err != nil {
		return pagination.PaginationParams{}, err
	}

	return pagination.NewPaginationParams(page, limit), nil
}
