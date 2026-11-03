package blossom

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/gabrielmoura/nostr-relay-server/config"
	dbmodel "github.com/gabrielmoura/nostr-relay-server/infra/db"
	storedb "github.com/gabrielmoura/nostr-relay-server/internal/db"
)

var (
	ErrInvalidPlanID    = errors.New("o ID do plano é obrigatório")
	ErrInvalidPlanName  = errors.New("o nome do plano é obrigatório")
	ErrInvalidPlanScope = errors.New("o escopo do plano deve ser 'free' ou 'enabled_users'")
	ErrNegativeQuota    = errors.New("as cotas de armazenamento e egress não podem ser negativas")
	ErrPlanNotFound     = errors.New("plano não encontrado")
)

func ActorPubkey() string {
	if config.Cfg != nil && strings.TrimSpace(config.Cfg.AdminPubKey) != "" {
		return config.Cfg.AdminPubKey
	}
	if config.Cfg != nil {
		return config.Cfg.RelayInformation.PubKey
	}
	return ""
}

func UpsertPlan(ctx context.Context, plan dbmodel.BlossomPlan) (*dbmodel.BlossomPlan, error) {
	if storedb.DbQueries == nil {
		return nil, errors.New("database queries not initialized")
	}

	plan.ID = strings.TrimSpace(plan.ID)
	if plan.ID == "" {
		return nil, ErrInvalidPlanID
	}
	plan.Name = strings.TrimSpace(plan.Name)
	if plan.Name == "" {
		return nil, ErrInvalidPlanName
	}
	plan.Scope = strings.ToLower(strings.TrimSpace(plan.Scope))
	if plan.Scope != "free" && plan.Scope != "enabled_users" {
		return nil, ErrInvalidPlanScope
	}
	if (plan.StorageQuotaBytes.Valid && plan.StorageQuotaBytes.Int64 < 0) ||
		(plan.EgressQuotaBytes.Valid && plan.EgressQuotaBytes.Int64 < 0) {
		return nil, ErrNegativeQuota
	}
	if strings.TrimSpace(plan.UpdatedBy) == "" {
		plan.UpdatedBy = ActorPubkey()
	}

	if err := storedb.DbQueries.UpsertBlossomPlan(ctx, plan); err != nil {
		return nil, fmt.Errorf("upsert blossom plan: %w", err)
	}

	_ = RecordAudit(ctx, plan.UpdatedBy, "plan.upsert", "plan", plan.ID, map[string]string{"scope": plan.Scope})

	stored, ok, err := storedb.DbQueries.GetBlossomPlan(ctx, plan.ID)
	if err != nil {
		return nil, fmt.Errorf("retrieve updated blossom plan: %w", err)
	}
	if !ok {
		return &plan, nil
	}
	return &stored, nil
}

func GetPlan(ctx context.Context, id string) (*dbmodel.BlossomPlan, bool, error) {
	if storedb.DbQueries == nil {
		return nil, false, errors.New("database queries not initialized")
	}
	plan, ok, err := storedb.DbQueries.GetBlossomPlan(ctx, strings.TrimSpace(id))
	if err != nil {
		return nil, false, err
	}
	if !ok {
		return nil, false, nil
	}
	return &plan, true, nil
}

func ListPlans(ctx context.Context, scope string) ([]dbmodel.BlossomPlan, error) {
	if storedb.DbQueries == nil {
		return nil, errors.New("database queries not initialized")
	}
	return storedb.DbQueries.ListBlossomPlans(ctx, strings.TrimSpace(scope))
}

func DeletePlan(ctx context.Context, id string) error {
	if storedb.DbQueries == nil {
		return errors.New("database queries not initialized")
	}
	cleanID := strings.TrimSpace(id)
	if cleanID == "" {
		return ErrInvalidPlanID
	}
	if err := storedb.DbQueries.DeleteBlossomPlan(ctx, cleanID); err != nil {
		return err
	}
	_ = RecordAudit(ctx, ActorPubkey(), "plan.delete", "plan", cleanID, nil)
	return nil
}

func AssignPlan(ctx context.Context, planID, pubkey, assignedBy string) error {
	if storedb.DbQueries == nil {
		return errors.New("database queries not initialized")
	}
	cleanID := strings.TrimSpace(planID)
	cleanPubkey := strings.TrimSpace(pubkey)
	if cleanID == "" {
		return ErrInvalidPlanID
	}
	if cleanPubkey == "" {
		return errors.New("pubkey é obrigatória")
	}
	if strings.TrimSpace(assignedBy) == "" {
		assignedBy = ActorPubkey()
	}
	if err := storedb.DbQueries.AssignPlanToUser(ctx, cleanID, cleanPubkey, assignedBy); err != nil {
		return err
	}
	_ = RecordAudit(ctx, assignedBy, "plan.assign", "pubkey", cleanPubkey, map[string]string{"plan_id": cleanID})
	return nil
}

func UnassignPlan(ctx context.Context, planID, pubkey, actorPubkey string) error {
	if storedb.DbQueries == nil {
		return errors.New("database queries not initialized")
	}
	cleanID := strings.TrimSpace(planID)
	cleanPubkey := strings.TrimSpace(pubkey)
	if cleanID == "" {
		return ErrInvalidPlanID
	}
	if cleanPubkey == "" {
		return errors.New("pubkey é obrigatória")
	}
	if strings.TrimSpace(actorPubkey) == "" {
		actorPubkey = ActorPubkey()
	}
	if err := storedb.DbQueries.UnassignPlanFromUser(ctx, cleanID, cleanPubkey); err != nil {
		return err
	}
	_ = RecordAudit(ctx, actorPubkey, "plan.unassign", "pubkey", cleanPubkey, map[string]string{"plan_id": cleanID})
	return nil
}

func ListPlanAssignments(ctx context.Context, planID string) ([]dbmodel.BlossomPlanAssignment, error) {
	if storedb.DbQueries == nil {
		return nil, errors.New("database queries not initialized")
	}
	cleanID := strings.TrimSpace(planID)
	if cleanID == "" {
		return nil, ErrInvalidPlanID
	}
	return storedb.DbQueries.ListPlanAssignments(ctx, cleanID)
}
