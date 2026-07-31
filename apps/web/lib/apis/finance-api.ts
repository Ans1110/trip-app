import { request } from "../utils";

// Monetary fields stay as strings on the wire — the Go backend serializes
// shopspring.Decimal as JSON strings to preserve precision. UI code is
// expected to render them as-is or parse with a decimal library, never
// coerce to Number.

export type SplitStrategy = "equal" | "custom" | "percentage";
export type SettlementStatus = "proposed" | "confirmed" | "cancelled";

export type ExpenseShare = {
  user_id: string;
  amount: string;
  amount_base: string;
  share_pct?: string;
  paid_at?: string;
};

export type SetSharePaidInput = {
  paid: boolean;
};

export type Expense = {
  id: string;
  trip_id: string;
  paid_by: string;
  amount: string;
  currency: string;
  amount_base: string;
  base_currency: string;
  description: string;
  category: string;
  split_strategy: SplitStrategy;
  occurred_at: string;
  receipt_asset_id?: string;
  created_by: string;
  created_at: string;
  updated_at: string;
  shares: ExpenseShare[];
};

export type Budget = {
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

export type Settlement = {
  id: string;
  trip_id: string;
  payer_id: string;
  payee_id: string;
  amount: string;
  currency: string;
  status: SettlementStatus;
  note: string;
  created_by: string;
  created_at: string;
  confirmed_at?: string;
  cancelled_at?: string;
};

export type CategoryStat = {
  category: string;
  amount_base: string;
  count: number;
};

export type PersonalStats = {
  trip_id: string;
  user_id: string;
  base_currency: string;
  total_paid: string;
  total_owed: string;
  net_balance: string;
  by_category: CategoryStat[];
};

export type TripBalance = {
  user_id: string;
  paid: string;
  owed: string;
  net: string;
};

export type ShareInput = {
  user_id: string;
  amount?: string;
  pct?: string;
};

export type CreateExpenseInput = {
  paid_by: string;
  amount: string;
  currency: string;
  description?: string;
  category?: string;
  split_strategy: SplitStrategy;
  participants: string[];
  shares?: ShareInput[];
  occurred_at?: string;
};

export type UpdateExpenseInput = {
  paid_by?: string;
  amount?: string;
  currency?: string;
  description?: string;
  category?: string;
  split_strategy?: SplitStrategy;
  participants?: string[];
  shares?: ShareInput[];
  occurred_at?: string;
};

export type UpsertBudgetInput = {
  category: string;
  amount: string;
  currency: string;
};

export type ConfirmSettlementInput = {
  note?: string;
};

export type ManualSettlementIn = {
  payer_id: string;
  payee_id: string;
  amount: string;
  note?: string;
};

export type ProposeSettlementInput = {
  auto: boolean;
  manual?: ManualSettlementIn[];
};

export type ExportCsvQuery = {
  from?: string;
  to?: string;
};

const path = (parts: string[]) => `/api/finance/${parts.join("/")}`;

export const financeApi = {
  // ---- expenses ----
  listExpenses: (tripId: string, signal?: AbortSignal) =>
    request<Expense[]>(
      path(["trips", encodeURIComponent(tripId), "expenses"]),
      {
        signal,
      },
    ),
  createExpense: (
    tripId: string,
    input: CreateExpenseInput,
    signal?: AbortSignal,
  ) =>
    request<Expense>(path(["trips", encodeURIComponent(tripId), "expenses"]), {
      method: "POST",
      json: input,
      signal,
    }),
  getExpense: (expenseId: string, signal?: AbortSignal) =>
    request<Expense>(path(["expenses", encodeURIComponent(expenseId)]), {
      signal,
    }),
  updateExpense: (
    expenseId: string,
    input: UpdateExpenseInput,
    signal?: AbortSignal,
  ) =>
    request<Expense>(path(["expenses", encodeURIComponent(expenseId)]), {
      method: "PATCH",
      json: input,
      signal,
    }),
  deleteExpense: (expenseId: string, signal?: AbortSignal) =>
    request<null>(path(["expenses", encodeURIComponent(expenseId)]), {
      method: "DELETE",
      signal,
    }),
  setSharePaid: (
    expenseId: string,
    userId: string,
    input: SetSharePaidInput,
    signal?: AbortSignal,
  ) =>
    request<Expense>(
      path([
        "expenses",
        encodeURIComponent(expenseId),
        "shares",
        encodeURIComponent(userId),
        "paid",
      ]),
      { method: "POST", json: input, signal },
    ),

  // ---- budgets ----
  listBudgets: (tripId: string, signal?: AbortSignal) =>
    request<Budget[]>(path(["trips", encodeURIComponent(tripId), "budgets"]), {
      signal,
    }),
  upsertBudget: (
    tripId: string,
    input: UpsertBudgetInput,
    signal?: AbortSignal,
  ) =>
    request<Budget>(path(["trips", encodeURIComponent(tripId), "budgets"]), {
      method: "POST",
      json: input,
      signal,
    }),
  deleteBudget: (tripId: string, budgetId: string, signal?: AbortSignal) =>
    request<null>(
      path([
        "trips",
        encodeURIComponent(tripId),
        "budgets",
        encodeURIComponent(budgetId),
      ]),
      { method: "DELETE", signal },
    ),

  // ---- settlements ----
  listSettlements: (tripId: string, signal?: AbortSignal) =>
    request<Settlement[]>(
      path(["trips", encodeURIComponent(tripId), "settlements"]),
      { signal },
    ),
  proposeSettlements: (
    tripId: string,
    input: ProposeSettlementInput,
    signal?: AbortSignal,
  ) =>
    request<Settlement[]>(
      path(["trips", encodeURIComponent(tripId), "settlements"]),
      { method: "POST", json: input, signal },
    ),
  confirmSettlement: (
    settlementId: string,
    input: ConfirmSettlementInput | null,
    signal?: AbortSignal,
  ) =>
    request<Settlement>(
      path(["settlements", encodeURIComponent(settlementId), "confirm"]),
      { method: "POST", json: input ?? {}, signal },
    ),
  cancelSettlement: (settlementId: string, signal?: AbortSignal) =>
    request<Settlement>(
      path(["settlements", encodeURIComponent(settlementId), "cancel"]),
      { method: "POST", signal },
    ),

  // ---- stats + export ----
  getTripBalances: (tripId: string, signal?: AbortSignal) =>
    request<TripBalance[]>(
      path(["trips", encodeURIComponent(tripId), "balances"]),
      { signal },
    ),
  getPersonalStats: (tripId: string, signal?: AbortSignal) =>
    request<PersonalStats>(
      path(["trips", encodeURIComponent(tripId), "stats", "me"]),
      { signal },
    ),
  exportCsvUrl: (tripId: string, query: ExportCsvQuery) => {
    const qs = new URLSearchParams();
    if (query.from) qs.set("from", query.from);
    if (query.to) qs.set("to", query.to);
    const suffix = qs.toString() ? `?${qs.toString()}` : "";
    return path(["trips", encodeURIComponent(tripId), "export.csv"]) + suffix;
  },
};
