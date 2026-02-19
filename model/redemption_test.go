package model

import (
	"fmt"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func setupRedemptionTestDB(t *testing.T) func() {
	t.Helper()

	oldDB := DB
	oldLogDB := LOG_DB
	oldUsingSQLite := common.UsingSQLite
	oldUsingMySQL := common.UsingMySQL
	oldUsingPostgreSQL := common.UsingPostgreSQL
	oldRedisEnabled := common.RedisEnabled
	oldGroupCol := commonGroupCol
	oldKeyCol := commonKeyCol
	oldTrueVal := commonTrueVal
	oldFalseVal := commonFalseVal

	common.UsingSQLite = true
	common.UsingMySQL = false
	common.UsingPostgreSQL = false
	common.RedisEnabled = false
	commonGroupCol = "`group`"
	commonKeyCol = "`key`"
	commonTrueVal = "1"
	commonFalseVal = "0"

	dsn := fmt.Sprintf("file:redemption_test_%d?mode=memory&cache=shared", time.Now().UnixNano())
	testDB, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open test db failed: %v", err)
	}
	DB = testDB
	LOG_DB = testDB

	if err := DB.AutoMigrate(&User{}, &Redemption{}, &SubscriptionPlan{}, &UserSubscription{}); err != nil {
		t.Fatalf("auto migrate failed: %v", err)
	}

	return func() {
		if sqlDB, err := testDB.DB(); err == nil {
			_ = sqlDB.Close()
		}
		DB = oldDB
		LOG_DB = oldLogDB
		common.UsingSQLite = oldUsingSQLite
		common.UsingMySQL = oldUsingMySQL
		common.UsingPostgreSQL = oldUsingPostgreSQL
		common.RedisEnabled = oldRedisEnabled
		commonGroupCol = oldGroupCol
		commonKeyCol = oldKeyCol
		commonTrueVal = oldTrueVal
		commonFalseVal = oldFalseVal
	}
}

func createTestUser(t *testing.T, quota int) *User {
	t.Helper()
	u := &User{
		Username:    fmt.Sprintf("redeem_test_%d", time.Now().UnixNano()),
		Password:    "12345678",
		DisplayName: "redeem-test",
		Role:        common.RoleCommonUser,
		Status:      common.UserStatusEnabled,
		Group:       "default",
		Quota:       quota,
	}
	if err := DB.Create(u).Error; err != nil {
		t.Fatalf("create user failed: %v", err)
	}
	return u
}

func TestRedeemSubscriptionPlan(t *testing.T) {
	cleanup := setupRedemptionTestDB(t)
	defer cleanup()

	user := createTestUser(t, 100)
	plan := &SubscriptionPlan{
		Title:       "Pro Plan",
		PriceAmount: 9.9,
		Currency:    "USD",
		Enabled:     true,
		TotalAmount: 1000000,
	}
	if err := DB.Create(plan).Error; err != nil {
		t.Fatalf("create plan failed: %v", err)
	}

	redemption := &Redemption{
		UserId:      user.Id,
		Key:         "sub-redemption-key-0000000000001",
		Status:      common.RedemptionCodeStatusEnabled,
		Name:        "sub",
		PlanId:      plan.Id,
		Quota:       0,
		CreatedTime: common.GetTimestamp(),
	}
	if err := DB.Create(redemption).Error; err != nil {
		t.Fatalf("create redemption failed: %v", err)
	}

	result, err := Redeem(redemption.Key, user.Id)
	if err != nil {
		t.Fatalf("redeem failed: %v", err)
	}
	if result.PlanId != plan.Id {
		t.Fatalf("expected plan id %d, got %d", plan.Id, result.PlanId)
	}
	if result.PlanTitle != plan.Title {
		t.Fatalf("expected plan title %q, got %q", plan.Title, result.PlanTitle)
	}
	if result.Quota != 0 {
		t.Fatalf("expected quota 0, got %d", result.Quota)
	}

	var sub UserSubscription
	if err := DB.Where("user_id = ? AND plan_id = ?", user.Id, plan.Id).First(&sub).Error; err != nil {
		t.Fatalf("expected user subscription created, query failed: %v", err)
	}
	if sub.Source != "redemption" {
		t.Fatalf("expected source redemption, got %s", sub.Source)
	}

	var updatedUser User
	if err := DB.First(&updatedUser, "id = ?", user.Id).Error; err != nil {
		t.Fatalf("query user failed: %v", err)
	}
	if updatedUser.Quota != 100 {
		t.Fatalf("expected user quota unchanged 100, got %d", updatedUser.Quota)
	}
}

func TestRedeemWalletQuota(t *testing.T) {
	cleanup := setupRedemptionTestDB(t)
	defer cleanup()

	user := createTestUser(t, 100)
	redemption := &Redemption{
		UserId:      user.Id,
		Key:         "quota-redemption-key-000000000001",
		Status:      common.RedemptionCodeStatusEnabled,
		Name:        "quota",
		Quota:       500,
		CreatedTime: common.GetTimestamp(),
	}
	if err := DB.Create(redemption).Error; err != nil {
		t.Fatalf("create redemption failed: %v", err)
	}

	result, err := Redeem(redemption.Key, user.Id)
	if err != nil {
		t.Fatalf("redeem failed: %v", err)
	}
	if result.Quota != 500 {
		t.Fatalf("expected redeemed quota 500, got %d", result.Quota)
	}
	if result.PlanId != 0 {
		t.Fatalf("expected plan id 0, got %d", result.PlanId)
	}

	var updatedUser User
	if err := DB.First(&updatedUser, "id = ?", user.Id).Error; err != nil {
		t.Fatalf("query user failed: %v", err)
	}
	if updatedUser.Quota != 600 {
		t.Fatalf("expected user quota 600, got %d", updatedUser.Quota)
	}
}
