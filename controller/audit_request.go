package controller

import (
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"

	"github.com/gin-gonic/gin"
)

func GetAuditRequestLogs(c *gin.Context) {
	pageInfo := common.GetPageQuery(c)
	startTimestamp, _ := strconv.ParseInt(c.Query("start_timestamp"), 10, 64)
	endTimestamp, _ := strconv.ParseInt(c.Query("end_timestamp"), 10, 64)

	query := model.AuditRequestLogQuery{
		StartTimestamp: startTimestamp,
		EndTimestamp:   endTimestamp,
		Ip:             c.Query("ip"),
		Path:           c.Query("path"),
		Method:         c.Query("method"),
		Username:       c.Query("username"),
		RequestId:      c.Query("request_id"),
		StartIdx:       pageInfo.GetStartIdx(),
		Num:            pageInfo.GetPageSize(),
	}
	for _, raw := range strings.Split(c.Query("status_code"), ",") {
		raw = strings.TrimSpace(raw)
		if raw == "" {
			continue
		}
		if statusCode, err := strconv.Atoi(raw); err == nil {
			query.StatusCodes = append(query.StatusCodes, statusCode)
		}
	}
	if result, err := strconv.Atoi(c.Query("result")); err == nil {
		query.Result = result
	}
	if raw := c.Query("user_id"); raw != "" {
		if userId, err := strconv.Atoi(raw); err == nil {
			query.UserId = &userId
		}
	}

	logs, total, err := model.GetAuditRequestLogs(query)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(logs)
	common.ApiSuccess(c, pageInfo)
}

func GetAuditRequestStatusCodes(c *gin.Context) {
	codes, err := model.GetAuditRequestStatusCodes()
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, codes)
}

func GetAuditRequestLogDetail(c *gin.Context) {
	requestId := c.Query("request_id")
	createdAt, _ := strconv.ParseInt(c.Query("created_at"), 10, 64)
	log, err := model.GetAuditRequestLogDetail(requestId, createdAt)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, log)
}
