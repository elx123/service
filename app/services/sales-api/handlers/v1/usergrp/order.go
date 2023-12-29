package usergrp

import (
	"errors"
	"net/http"

	"github.com/ardanlabs/service/business/core/user"
	"github.com/ardanlabs/service/business/cview/user/summary"
	"github.com/ardanlabs/service/business/data/order"
	"github.com/ardanlabs/service/business/sys/validate"
)

var orderByFields = map[string]struct{}{
	user.OrderByID:      {},
	user.OrderByName:    {},
	user.OrderByEmail:   {},
	user.OrderByRoles:   {},
	user.OrderByEnabled: {},
}

// 这里我们要根据application lvl 的string 构造一个business lvl 的order by
func parseOrder(r *http.Request) (order.By, error) {
	orderBy, err := order.Parse(r, user.DefaultOrderBy)
	if err != nil {
		return order.By{}, err
	}

	// 这里我们将生成的orderBy 与business层 做一个校验
	if _, exists := orderByFields[orderBy.Field]; !exists {
		return order.By{}, validate.NewFieldsError(orderBy.Field, errors.New("order field does not exist"))
	}

	return orderBy, nil
}

// =============================================================================

var orderBySummaryFields = map[string]struct{}{
	summary.OrderByUserID:   {},
	summary.OrderByUserName: {},
}

func parseSummaryOrder(r *http.Request) (order.By, error) {
	orderBy, err := order.Parse(r, user.DefaultOrderBy)
	if err != nil {
		return order.By{}, err
	}

	if _, exists := orderBySummaryFields[orderBy.Field]; !exists {
		return order.By{}, validate.NewFieldsError(orderBy.Field, errors.New("order field does not exist"))
	}

	return orderBy, nil
}
