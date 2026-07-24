import "server-only";
import { circuitOpenError } from "../errors";
import {
  ApiErr,
  ApiResult,
  Envelope,
  httpClient,
  HttpClient,
  HttpOptions,
} from "../http-client";
import { RequestContext } from "../request-context";
import { ensureCsrfToken, readSessionTokens } from "../session-store";
import {
  BudgetView,
  ExpenseView,
  FxRateView,
  PersonalStatsView,
  SettlementView,
  toBudgetView,
  toExpenseView,
  toFxRateView,
  toPersonalStatsView,
  toSettlementView,
  toTripBalanceView,
  TripBalanceView,
} from "../contracts/frontend";
import {
  UpstreamBudget,
  UpstreamConfirmSettlementPayload,
  UpstreamCreateExpensePayload,
  UpstreamExpense,
  UpstreamExportCsvQuery,
  UpstreamFxRate,
  UpstreamListFxRatesQuery,
  UpstreamPersonalStats,
  UpstreamProposeSettlementPayload,
  UpstreamSettlement,
  UpstreamTripBalance,
  UpstreamUpdateExpensePayload,
  UpstreamUpsertBudgetPayload,
  UpstreamUpsertFxRatePayload,
} from "../contracts/upstream";

export type CreateExpenseInput = UpstreamCreateExpensePayload;
export type UpdateExpenseInput = UpstreamUpdateExpensePayload;
export type UpsertBudgetInput = UpstreamUpsertBudgetPayload;
export type UpsertFxRateInput = UpstreamUpsertFxRatePayload;
export type ConfirmSettlementInput = UpstreamConfirmSettlementPayload;
export type ProposeSettlementInput = UpstreamProposeSettlementPayload;
export type ListFxRatesQuery = UpstreamListFxRatesQuery;
export type ExportCsvQuery = UpstreamExportCsvQuery;

export type {
  BudgetView,
  CategoryStatView,
  ExpenseShareView,
  ExpenseView,
  FxRateView,
  PersonalStatsView,
  SettlementStatusView,
  SettlementView,
  SplitStrategyView,
  TripBalanceView,
} from "../contracts/frontend";

type AuthCtx = { ctx: RequestContext; signal?: AbortSignal };

type ProtectedDeps = {
  accessToken: string;
  csrfToken: string;
};

const MISSING_SESSION = unauthorizedError("missing session");
const MISSING_CSRF = circuitGuardError("missing CSRF token");

// FinanceClient proxies expense / budget / settlement / FX operations for the
// Go finance service. CSV export is returned as ApiResult<string> — the raw
// body is surfaced through `data` so the route handler can stream it back
// with a text/csv content-type.
export class FinanceClient {
  constructor(private readonly http: HttpClient = httpClient) {}

  // ---- expenses ----

  async listExpenses(
    tripID: string,
    auth: AuthCtx,
  ): Promise<ApiResult<ExpenseView[]>> {
    const res = await this.callProtected<UpstreamExpense[]>(auth, (opts) =>
      this.http.get<UpstreamExpense[]>(
        `/trips/${encodeURIComponent(tripID)}/finance/expenses`,
        opts,
      ),
    );
    return mapData(res, (rows) => (rows ?? []).map(toExpenseView));
  }

  async createExpense(
    tripID: string,
    input: CreateExpenseInput,
    auth: AuthCtx,
  ): Promise<ApiResult<ExpenseView>> {
    const res = await this.callProtected<UpstreamExpense>(auth, (opts) =>
      this.http.post<UpstreamExpense>(
        `/trips/${encodeURIComponent(tripID)}/finance/expenses`,
        input,
        opts,
      ),
    );
    return mapData(res, toExpenseView);
  }

  async getExpense(
    expenseID: string,
    auth: AuthCtx,
  ): Promise<ApiResult<ExpenseView>> {
    const res = await this.callProtected<UpstreamExpense>(auth, (opts) =>
      this.http.get<UpstreamExpense>(
        `/finance/expenses/${encodeURIComponent(expenseID)}`,
        opts,
      ),
    );
    return mapData(res, toExpenseView);
  }

  async updateExpense(
    expenseID: string,
    input: UpdateExpenseInput,
    auth: AuthCtx,
  ): Promise<ApiResult<ExpenseView>> {
    const res = await this.callProtected<UpstreamExpense>(auth, (opts) =>
      this.http.patch<UpstreamExpense>(
        `/finance/expenses/${encodeURIComponent(expenseID)}`,
        input,
        opts,
      ),
    );
    return mapData(res, toExpenseView);
  }

  async deleteExpense(
    expenseID: string,
    auth: AuthCtx,
  ): Promise<ApiResult<null>> {
    return this.callProtected<null>(auth, (opts) =>
      this.http.delete<null>(
        `/finance/expenses/${encodeURIComponent(expenseID)}`,
        opts,
      ),
    );
  }

  // ---- budgets ----

  async listBudgets(
    tripID: string,
    auth: AuthCtx,
  ): Promise<ApiResult<BudgetView[]>> {
    const res = await this.callProtected<UpstreamBudget[]>(auth, (opts) =>
      this.http.get<UpstreamBudget[]>(
        `/trips/${encodeURIComponent(tripID)}/finance/budgets`,
        opts,
      ),
    );
    return mapData(res, (rows) => (rows ?? []).map(toBudgetView));
  }

  async upsertBudget(
    tripID: string,
    input: UpsertBudgetInput,
    auth: AuthCtx,
  ): Promise<ApiResult<BudgetView>> {
    const res = await this.callProtected<UpstreamBudget>(auth, (opts) =>
      this.http.post<UpstreamBudget>(
        `/trips/${encodeURIComponent(tripID)}/finance/budgets`,
        input,
        opts,
      ),
    );
    return mapData(res, toBudgetView);
  }

  async deleteBudget(
    tripID: string,
    budgetID: string,
    auth: AuthCtx,
  ): Promise<ApiResult<null>> {
    return this.callProtected<null>(auth, (opts) =>
      this.http.delete<null>(
        `/trips/${encodeURIComponent(tripID)}/finance/budgets/${encodeURIComponent(budgetID)}`,
        opts,
      ),
    );
  }

  // ---- settlements ----

  async listSettlements(
    tripID: string,
    auth: AuthCtx,
  ): Promise<ApiResult<SettlementView[]>> {
    const res = await this.callProtected<UpstreamSettlement[]>(auth, (opts) =>
      this.http.get<UpstreamSettlement[]>(
        `/trips/${encodeURIComponent(tripID)}/finance/settlements`,
        opts,
      ),
    );
    return mapData(res, (rows) => (rows ?? []).map(toSettlementView));
  }

  async proposeSettlements(
    tripID: string,
    input: ProposeSettlementInput,
    auth: AuthCtx,
  ): Promise<ApiResult<SettlementView[]>> {
    const res = await this.callProtected<UpstreamSettlement[]>(auth, (opts) =>
      this.http.post<UpstreamSettlement[]>(
        `/trips/${encodeURIComponent(tripID)}/finance/settlements`,
        input,
        opts,
      ),
    );
    return mapData(res, (rows) => (rows ?? []).map(toSettlementView));
  }

  async confirmSettlement(
    settlementID: string,
    input: ConfirmSettlementInput | null,
    auth: AuthCtx,
  ): Promise<ApiResult<SettlementView>> {
    const res = await this.callProtected<UpstreamSettlement>(auth, (opts) =>
      this.http.post<UpstreamSettlement>(
        `/finance/settlements/${encodeURIComponent(settlementID)}/confirm`,
        input,
        opts,
      ),
    );
    return mapData(res, toSettlementView);
  }

  async cancelSettlement(
    settlementID: string,
    auth: AuthCtx,
  ): Promise<ApiResult<SettlementView>> {
    const res = await this.callProtected<UpstreamSettlement>(auth, (opts) =>
      this.http.post<UpstreamSettlement>(
        `/finance/settlements/${encodeURIComponent(settlementID)}/cancel`,
        null,
        opts,
      ),
    );
    return mapData(res, toSettlementView);
  }

  // ---- stats + export ----

  async getTripBalances(
    tripID: string,
    auth: AuthCtx,
  ): Promise<ApiResult<TripBalanceView[]>> {
    const res = await this.callProtected<UpstreamTripBalance[]>(auth, (opts) =>
      this.http.get<UpstreamTripBalance[]>(
        `/trips/${encodeURIComponent(tripID)}/finance/balances`,
        opts,
      ),
    );
    return mapData(res, (rows) => (rows ?? []).map(toTripBalanceView));
  }

  async getPersonalStats(
    tripID: string,
    auth: AuthCtx,
  ): Promise<ApiResult<PersonalStatsView>> {
    const res = await this.callProtected<UpstreamPersonalStats>(auth, (opts) =>
      this.http.get<UpstreamPersonalStats>(
        `/trips/${encodeURIComponent(tripID)}/finance/stats/me`,
        opts,
      ),
    );
    return mapData(res, toPersonalStatsView);
  }

  // ExportCSV surfaces the raw response body as data because http-client's
  // JSON parse gracefully yields null for non-JSON payloads — the CSV text
  // still lives in the `raw` field of the result.
  async exportCSV(
    tripID: string,
    query: ExportCsvQuery,
    auth: AuthCtx,
  ): Promise<ApiResult<string>> {
    const qs = new URLSearchParams();
    if (query.from) qs.set("from", query.from);
    if (query.to) qs.set("to", query.to);
    const suffix = qs.toString() ? `?${qs.toString()}` : "";
    const res = await this.callProtected<unknown>(auth, (opts) =>
      this.http.get<unknown>(
        `/trips/${encodeURIComponent(tripID)}/finance/export.csv${suffix}`,
        opts,
      ),
    );
    if (!res.ok) return res as ApiResult<string>;
    return {
      ...res,
      data: res.raw,
      envelope: undefined,
    };
  }

  // ---- fx ----

  async listFxRates(
    query: ListFxRatesQuery,
    auth: AuthCtx,
  ): Promise<ApiResult<FxRateView[]>> {
    const suffix = query.base
      ? `?base=${encodeURIComponent(query.base)}`
      : "";
    const res = await this.callProtected<UpstreamFxRate[]>(auth, (opts) =>
      this.http.get<UpstreamFxRate[]>(`/finance/fx${suffix}`, opts),
    );
    return mapData(res, (rows) => (rows ?? []).map(toFxRateView));
  }

  async upsertFxRate(
    input: UpsertFxRateInput,
    auth: AuthCtx,
  ): Promise<ApiResult<FxRateView>> {
    const res = await this.callProtected<UpstreamFxRate>(auth, (opts) =>
      this.http.post<UpstreamFxRate>("/finance/fx", input, opts),
    );
    return mapData(res, toFxRateView);
  }

  // ---- plumbing ----

  private protectedOpts(auth: AuthCtx, deps: ProtectedDeps): HttpOptions {
    return {
      ctx: auth.ctx,
      signal: auth.signal,
      accessToken: deps.accessToken,
      csrfToken: deps.csrfToken,
    };
  }

  private async resolveProtected(
    auth: AuthCtx,
  ): Promise<{ deps: ProtectedDeps } | { error: ApiErr }> {
    const tokens = await readSessionTokens();
    if (!tokens.accessToken) return { error: MISSING_SESSION };
    const csrf = tokens.csrfToken ?? (await ensureCsrfToken(auth.ctx)) ?? null;
    if (!csrf) return { error: MISSING_CSRF };
    return {
      deps: {
        accessToken: tokens.accessToken,
        csrfToken: csrf,
      },
    };
  }

  private async callProtected<T>(
    auth: AuthCtx,
    invoke: (opts: HttpOptions) => Promise<ApiResult<T>>,
  ): Promise<ApiResult<T>> {
    const resolved = await this.resolveProtected(auth);
    if ("error" in resolved) return resolved.error as ApiResult<T>;
    return invoke(this.protectedOpts(auth, resolved.deps));
  }
}

const mapData = <TIn, TOut>(
  result: ApiResult<TIn>,
  map: (v: TIn) => TOut,
): ApiResult<TOut> => {
  if (!result.ok) return result;
  return {
    ...result,
    data: map(result.data),
    envelope: result.envelope as unknown as Envelope<TOut> | undefined,
  };
};

function unauthorizedError(message: string): ApiErr {
  return {
    ok: false,
    status: 401,
    code: 401,
    message,
    error: {
      category: "auth",
      status: 401,
      code: 401,
      message,
      retryable: false,
    },
    payload: null,
    raw: "",
  };
}

function circuitGuardError(message: string): ApiErr {
  const base = circuitOpenError();
  return {
    ok: false,
    status: base.status,
    code: base.code,
    message,
    error: { ...base, message },
    payload: null,
    raw: "",
  };
}

export const financeClient = new FinanceClient();
