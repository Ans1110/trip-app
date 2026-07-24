import "server-only";
import { NextRequest, NextResponse } from "next/server";

import type {
  ConfirmSettlementInput,
  CreateExpenseInput,
  ProposeSettlementInput,
  UpdateExpenseInput,
  UpsertBudgetInput,
  UpsertFxRateInput,
} from "../../_lib/clients/finance-client";
import { financeClient } from "../../_lib/clients/finance-client";
import type { ApiResult } from "../../_lib/http-client";
import { extractRequestContext } from "../../_lib/request-context";

type RouteParams = { params: Promise<{ slug?: string[] }> };

type AuthArg = {
  ctx: ReturnType<typeof extractRequestContext>;
  signal?: AbortSignal;
};

type Dispatch = (req: NextRequest, auth: AuthArg) => Promise<NextResponse>;

export async function GET(req: NextRequest, { params }: RouteParams) {
  const slug = (await params).slug ?? [];
  const dispatch = matchGet(slug);
  if (!dispatch) return notFound(req.method, slug);
  return dispatch(req, buildAuth(req));
}

export async function POST(req: NextRequest, { params }: RouteParams) {
  const slug = (await params).slug ?? [];
  const dispatch = matchPost(slug);
  if (!dispatch) return notFound(req.method, slug);
  return dispatch(req, buildAuth(req));
}

export async function PATCH(req: NextRequest, { params }: RouteParams) {
  const slug = (await params).slug ?? [];
  const dispatch = matchPatch(slug);
  if (!dispatch) return notFound(req.method, slug);
  return dispatch(req, buildAuth(req));
}

export async function DELETE(req: NextRequest, { params }: RouteParams) {
  const slug = (await params).slug ?? [];
  const dispatch = matchDelete(slug);
  if (!dispatch) return notFound(req.method, slug);
  return dispatch(req, buildAuth(req));
}

const matchGet = (slug: string[]): Dispatch | null => {
  // GET /finance/trips/:tripID/expenses
  if (slug.length === 3 && slug[0] === "trips" && slug[2] === "expenses") {
    return async (_req, auth) =>
      toResponse(await financeClient.listExpenses(slug[1], auth));
  }
  // GET /finance/trips/:tripID/budgets
  if (slug.length === 3 && slug[0] === "trips" && slug[2] === "budgets") {
    return async (_req, auth) =>
      toResponse(await financeClient.listBudgets(slug[1], auth));
  }
  // GET /finance/trips/:tripID/settlements
  if (slug.length === 3 && slug[0] === "trips" && slug[2] === "settlements") {
    return async (_req, auth) =>
      toResponse(await financeClient.listSettlements(slug[1], auth));
  }
  // GET /finance/trips/:tripID/balances
  if (slug.length === 3 && slug[0] === "trips" && slug[2] === "balances") {
    return async (_req, auth) =>
      toResponse(await financeClient.getTripBalances(slug[1], auth));
  }
  // GET /finance/trips/:tripID/stats/me
  if (
    slug.length === 4 &&
    slug[0] === "trips" &&
    slug[2] === "stats" &&
    slug[3] === "me"
  ) {
    return async (_req, auth) =>
      toResponse(await financeClient.getPersonalStats(slug[1], auth));
  }
  // GET /finance/trips/:tripID/export.csv
  if (slug.length === 3 && slug[0] === "trips" && slug[2] === "export.csv") {
    return async (req, auth) => {
      const from = req.nextUrl.searchParams.get("from") ?? undefined;
      const to = req.nextUrl.searchParams.get("to") ?? undefined;
      const result = await financeClient.exportCSV(slug[1], { from, to }, auth);
      if (!result.ok) return toResponse(result);
      return csvResponse(slug[1], result.data);
    };
  }
  // GET /finance/expenses/:expenseID
  if (slug.length === 2 && slug[0] === "expenses") {
    return async (_req, auth) =>
      toResponse(await financeClient.getExpense(slug[1], auth));
  }
  // GET /finance/fx
  if (slug.length === 1 && slug[0] === "fx") {
    return async (req, auth) => {
      const base = req.nextUrl.searchParams.get("base") ?? undefined;
      return toResponse(await financeClient.listFxRates({ base }, auth));
    };
  }
  return null;
};

const matchPost = (slug: string[]): Dispatch | null => {
  // POST /finance/trips/:tripID/expenses
  if (slug.length === 3 && slug[0] === "trips" && slug[2] === "expenses") {
    return async (req, auth) => {
      const body = await readJson(req);
      if (body instanceof NextResponse) return body;
      return toResponse(
        await financeClient.createExpense(
          slug[1],
          body as CreateExpenseInput,
          auth,
        ),
      );
    };
  }
  // POST /finance/trips/:tripID/budgets
  if (slug.length === 3 && slug[0] === "trips" && slug[2] === "budgets") {
    return async (req, auth) => {
      const body = await readJson(req);
      if (body instanceof NextResponse) return body;
      return toResponse(
        await financeClient.upsertBudget(
          slug[1],
          body as UpsertBudgetInput,
          auth,
        ),
      );
    };
  }
  // POST /finance/trips/:tripID/settlements
  if (slug.length === 3 && slug[0] === "trips" && slug[2] === "settlements") {
    return async (req, auth) => {
      const body = await readJson(req);
      if (body instanceof NextResponse) return body;
      return toResponse(
        await financeClient.proposeSettlements(
          slug[1],
          body as ProposeSettlementInput,
          auth,
        ),
      );
    };
  }
  // POST /finance/settlements/:settlementID/confirm
  if (
    slug.length === 3 &&
    slug[0] === "settlements" &&
    slug[2] === "confirm"
  ) {
    return async (req, auth) => {
      const body = await readJson(req);
      if (body instanceof NextResponse) return body;
      return toResponse(
        await financeClient.confirmSettlement(
          slug[1],
          body as ConfirmSettlementInput | null,
          auth,
        ),
      );
    };
  }
  // POST /finance/settlements/:settlementID/cancel
  if (slug.length === 3 && slug[0] === "settlements" && slug[2] === "cancel") {
    return async (_req, auth) =>
      toResponse(await financeClient.cancelSettlement(slug[1], auth));
  }
  // POST /finance/fx
  if (slug.length === 1 && slug[0] === "fx") {
    return async (req, auth) => {
      const body = await readJson(req);
      if (body instanceof NextResponse) return body;
      return toResponse(
        await financeClient.upsertFxRate(body as UpsertFxRateInput, auth),
      );
    };
  }
  return null;
};

const matchPatch = (slug: string[]): Dispatch | null => {
  // PATCH /finance/expenses/:expenseID
  if (slug.length === 2 && slug[0] === "expenses") {
    return async (req, auth) => {
      const body = await readJson(req);
      if (body instanceof NextResponse) return body;
      return toResponse(
        await financeClient.updateExpense(
          slug[1],
          body as UpdateExpenseInput,
          auth,
        ),
      );
    };
  }
  return null;
};

const matchDelete = (slug: string[]): Dispatch | null => {
  // DELETE /finance/expenses/:expenseID
  if (slug.length === 2 && slug[0] === "expenses") {
    return async (_req, auth) =>
      toResponse(await financeClient.deleteExpense(slug[1], auth));
  }
  // DELETE /finance/trips/:tripID/budgets/:budgetID
  if (
    slug.length === 4 &&
    slug[0] === "trips" &&
    slug[2] === "budgets"
  ) {
    return async (_req, auth) =>
      toResponse(await financeClient.deleteBudget(slug[1], slug[3], auth));
  }
  return null;
};

const buildAuth = (req: NextRequest): AuthArg => ({
  ctx: extractRequestContext(req),
  signal: req.signal,
});

const readJson = async (req: NextRequest): Promise<unknown> => {
  const len = req.headers.get("content-length");
  if (!len || len === "0") return null;
  try {
    return await req.json();
  } catch {
    return NextResponse.json(
      { code: 400, message: "invalid json body" },
      { status: 400 },
    );
  }
};

const toResponse = (result: ApiResult<unknown>): NextResponse => {
  if (result.ok) {
    return NextResponse.json(
      {
        code: result.envelope?.code ?? result.status,
        message: result.envelope?.message ?? "ok",
        data: result.data ?? null,
      },
      { status: result.status },
    );
  }
  return NextResponse.json(
    {
      code: result.envelope?.code ?? result.code ?? result.status,
      message: result.message,
      error: {
        category: result.error.category,
        retryable: result.error.retryable,
      },
    },
    { status: result.status >= 400 ? result.status : 500 },
  );
};

const csvResponse = (tripID: string, csv: string): NextResponse => {
  return new NextResponse(csv, {
    status: 200,
    headers: {
      "content-type": "text/csv; charset=utf-8",
      "content-disposition": `attachment; filename="trip-${tripID}-finance.csv"`,
      "cache-control": "no-store",
    },
  });
};

const notFound = (method: string, slug: string[]): NextResponse =>
  NextResponse.json(
    {
      code: 404,
      message: `unknown finance op: ${method} ${slug.join("/")}`,
    },
    { status: 404 },
  );
