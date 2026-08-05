import "server-only";
import {
  UpstreamBudget,
  UpstreamCategoryStat,
  UpstreamExpense,
  UpstreamExpenseShare,
  UpstreamFxRate,
  UpstreamPersonalStats,
  UpstreamSettlement,
  UpstreamSettlementStatus,
  UpstreamSplitStrategy,
  UpstreamTripBalance,
} from "../upstream";

export type SplitStrategyView = UpstreamSplitStrategy;
export type SettlementStatusView = UpstreamSettlementStatus;

export type ExpenseShareView = {
  user_id: string;
  amount: string;
  amount_base: string;
  share_pct?: string;
  paid_at?: string;
};

export type ExpenseView = {
  id: string;
  trip_id: string;
  paid_by: string;
  amount: string;
  currency: string;
  amount_base: string;
  base_currency: string;
  rate_to_base?: string;
  description: string;
  category: string;
  split_strategy: SplitStrategyView;
  occurred_at: string;
  receipt_asset_id?: string;
  created_by: string;
  created_at: string;
  updated_at: string;
  shares: ExpenseShareView[];
};

export type BudgetView = {
  id: string;
  trip_id: string;
  category: string;
  amount: string;
  currency: string;
  spent_base: string;
  remaining_base: string;
  over_budget: boolean;
  created_by: string;
  created_at: string;
  updated_at: string;
};

export type SettlementView = {
  id: string;
  trip_id: string;
  payer_id: string;
  payee_id: string;
  amount: string;
  currency: string;
  status: SettlementStatusView;
  note: string;
  created_by: string;
  created_at: string;
  confirmed_at?: string;
  cancelled_at?: string;
};

export type FxRateView = {
  base: string;
  quote: string;
  rate: string;
  as_of: string;
  source: string;
};

export type CategoryStatView = {
  category: string;
  amount_base: string;
  count: number;
};

export type PersonalStatsView = {
  trip_id: string;
  user_id: string;
  base_currency: string;
  total_paid: string;
  total_owed: string;
  net_balance: string;
  by_category: CategoryStatView[];
};

export type TripBalanceView = {
  user_id: string;
  paid: string;
  owed: string;
  net: string;
};

export const toExpenseShareView = (
  s: UpstreamExpenseShare,
): ExpenseShareView => ({
  user_id: s.user_id,
  amount: s.amount,
  amount_base: s.amount_base,
  share_pct: s.share_pct,
  paid_at: s.paid_at,
});

export const toExpenseView = (e: UpstreamExpense): ExpenseView => ({
  id: e.id,
  trip_id: e.trip_id,
  paid_by: e.paid_by,
  amount: e.amount,
  currency: e.currency,
  amount_base: e.amount_base,
  base_currency: e.base_currency,
  rate_to_base: e.rate_to_base,
  description: e.description,
  category: e.category,
  split_strategy: e.split_strategy,
  occurred_at: e.occurred_at,
  receipt_asset_id: e.receipt_asset_id,
  created_by: e.created_by,
  created_at: e.created_at,
  updated_at: e.updated_at,
  shares: (e.shares ?? []).map(toExpenseShareView),
});

export const toBudgetView = (b: UpstreamBudget): BudgetView => ({
  id: b.id,
  trip_id: b.trip_id,
  category: b.category,
  amount: b.amount,
  currency: b.currency,
  spent_base: b.spent_base,
  remaining_base: b.remaining_base,
  over_budget: b.over_budget,
  created_by: b.created_by,
  created_at: b.created_at,
  updated_at: b.updated_at,
});

export const toSettlementView = (s: UpstreamSettlement): SettlementView => ({
  id: s.id,
  trip_id: s.trip_id,
  payer_id: s.payer_id,
  payee_id: s.payee_id,
  amount: s.amount,
  currency: s.currency,
  status: s.status,
  note: s.note,
  created_by: s.created_by,
  created_at: s.created_at,
  confirmed_at: s.confirmed_at,
  cancelled_at: s.cancelled_at,
});

export const toFxRateView = (r: UpstreamFxRate): FxRateView => ({
  base: r.base,
  quote: r.quote,
  rate: r.rate,
  as_of: r.as_of,
  source: r.source,
});

export const toCategoryStatView = (
  c: UpstreamCategoryStat,
): CategoryStatView => ({
  category: c.category,
  amount_base: c.amount_base,
  count: c.count ?? 0,
});

export const toPersonalStatsView = (
  s: UpstreamPersonalStats,
): PersonalStatsView => ({
  trip_id: s.trip_id,
  user_id: s.user_id,
  base_currency: s.base_currency,
  total_paid: s.total_paid,
  total_owed: s.total_owed,
  net_balance: s.net_balance,
  by_category: (s.by_category ?? []).map(toCategoryStatView),
});

export const toTripBalanceView = (b: UpstreamTripBalance): TripBalanceView => ({
  user_id: b.user_id,
  paid: b.paid,
  owed: b.owed,
  net: b.net,
});
