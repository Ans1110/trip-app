import "server-only";

// Go's shopspring/decimal marshals to JSON strings, so every monetary field is
// a decimal-formatted string ("12.50"), not a number.
export type UpstreamSplitStrategy = "equal" | "custom" | "percentage";
export type UpstreamSettlementStatus = "proposed" | "confirmed" | "cancelled";

export type UpstreamShareInput = {
  user_id: string;
  amount?: string;
  pct?: string;
};

export type UpstreamCreateExpensePayload = {
  paid_by: string;
  amount: string;
  currency: string;
  description?: string;
  category?: string;
  split_strategy: UpstreamSplitStrategy;
  participants: string[];
  shares?: UpstreamShareInput[];
  occurred_at?: string;
  receipt_asset_id?: string;
};

export type UpstreamUpdateExpensePayload = {
  paid_by?: string;
  amount?: string;
  currency?: string;
  description?: string;
  category?: string;
  split_strategy?: UpstreamSplitStrategy;
  participants?: string[];
  shares?: UpstreamShareInput[];
  occurred_at?: string;
  receipt_asset_id?: string;
  clear_receipt?: boolean;
};

export type UpstreamUpsertBudgetPayload = {
  category: string;
  amount: string;
  currency: string;
};

export type UpstreamUpsertFxRatePayload = {
  base: string;
  quote: string;
  rate: string;
  as_of?: string;
};

export type UpstreamConfirmSettlementPayload = {
  note?: string;
};

export type UpstreamManualSettlementIn = {
  payer_id: string;
  payee_id: string;
  amount: string;
  note?: string;
};

export type UpstreamProposeSettlementPayload = {
  auto: boolean;
  manual?: UpstreamManualSettlementIn[];
};

export type UpstreamExpenseShare = {
  user_id: string;
  amount: string;
  amount_base: string;
  share_pct?: string;
  paid_at?: string;
};

export type UpstreamSetSharePaidPayload = {
  paid: boolean;
};

export type UpstreamExpense = {
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
  split_strategy: UpstreamSplitStrategy;
  occurred_at: string;
  receipt_asset_id?: string;
  created_by: string;
  created_at: string;
  updated_at: string;
  shares: UpstreamExpenseShare[];
};

export type UpstreamBudget = {
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

export type UpstreamSettlement = {
  id: string;
  trip_id: string;
  payer_id: string;
  payee_id: string;
  amount: string;
  currency: string;
  status: UpstreamSettlementStatus;
  note: string;
  created_by: string;
  created_at: string;
  confirmed_at?: string;
  cancelled_at?: string;
};

export type UpstreamFxRate = {
  base: string;
  quote: string;
  rate: string;
  as_of: string;
  source: string;
};

export type UpstreamCategoryStat = {
  category: string;
  amount_base: string;
  count: number;
};

export type UpstreamPersonalStats = {
  trip_id: string;
  user_id: string;
  base_currency: string;
  total_paid: string;
  total_owed: string;
  net_balance: string;
  by_category: UpstreamCategoryStat[];
};

export type UpstreamTripBalance = {
  user_id: string;
  paid: string;
  owed: string;
  net: string;
};

export type UpstreamListFxRatesQuery = {
  base?: string;
};

export type UpstreamExportCsvQuery = {
  from?: string;
  to?: string;
};
