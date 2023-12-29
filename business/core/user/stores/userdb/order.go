package userdb

import (
	"fmt"

	"github.com/ardanlabs/service/business/core/user"
	"github.com/ardanlabs/service/business/data/order"
)

var orderByFields = map[string]string{
	user.OrderByID:      "user_id",
	user.OrderByName:    "name",
	user.OrderByEmail:   "email",
	user.OrderByRoles:   "roles",
	user.OrderByEnabled: "enabled",
}

func orderByClause(orderBy order.By) (string, error) {
	// 这里有个细节,就是我们用business 层的ID,去映射store 层的ID
	by, exists := orderByFields[orderBy.Field]
	if !exists {
		return "", fmt.Errorf("field %q does not exist", orderBy.Field)
	}

	return " ORDER BY " + by + " " + orderBy.Direction, nil
}
