"use client";

import {
  useMutation,
  useQuery,
  useQueryClient,
  type UseMutationOptions,
  type UseQueryOptions,
} from "@tanstack/react-query";

import { ApiError } from "@/lib/utils";
import {
  financeApi,
  type Budget,
  type CreateExpenseInput,
  type Expense,
  type PersonalStats,
  type SetSharePaidInput,
  type UpsertBudgetInput,
} from "@/lib/apis/finance-api";

export { errorMessage } from "@/hooks/trip-hooks";

export const financeKeys = {
  all: ["finance"] as const,
  expenses: (tripId: string) => ["finance", "expenses", tripId] as const,
  budgets: (tripId: string) => ["finance", "budgets", tripId] as const,
  personalStats: (tripId: string) =>
    ["finance", "stats", "me", tripId] as const,
};

function invalidateTripFinance(
  qc: ReturnType<typeof useQueryClient>,
  tripId: string,
) {
  void qc.invalidateQueries({ queryKey: financeKeys.expenses(tripId) });
  void qc.invalidateQueries({ queryKey: financeKeys.budgets(tripId) });
  void qc.invalidateQueries({ queryKey: financeKeys.personalStats(tripId) });
}

type MutOpts<TData, TInput> = Omit<
  UseMutationOptions<TData, ApiError, TInput>,
  "mutationFn"
>;

type QueryOpts<TData> = Omit<
  UseQueryOptions<TData, ApiError>,
  "queryKey" | "queryFn"
>;

export function useExpenses(
  tripId: string | undefined,
  options?: QueryOpts<Expense[]>,
) {
  return useQuery<Expense[], ApiError>({
    queryKey: financeKeys.expenses(tripId ?? ""),
    queryFn: ({ signal }) => financeApi.listExpenses(tripId!, signal),
    enabled: !!tripId,
    ...options,
  });
}

export function useBudgets(
  tripId: string | undefined,
  options?: QueryOpts<Budget[]>,
) {
  return useQuery<Budget[], ApiError>({
    queryKey: financeKeys.budgets(tripId ?? ""),
    queryFn: ({ signal }) => financeApi.listBudgets(tripId!, signal),
    enabled: !!tripId,
    ...options,
  });
}

export function usePersonalStats(
  tripId: string | undefined,
  options?: QueryOpts<PersonalStats>,
) {
  return useQuery<PersonalStats, ApiError>({
    queryKey: financeKeys.personalStats(tripId ?? ""),
    queryFn: ({ signal }) => financeApi.getPersonalStats(tripId!, signal),
    enabled: !!tripId,
    ...options,
  });
}

type CreateExpenseVars = { tripId: string; input: CreateExpenseInput };

export function useCreateExpense(
  options?: MutOpts<Expense, CreateExpenseVars>,
) {
  const qc = useQueryClient();
  return useMutation<Expense, ApiError, CreateExpenseVars>({
    mutationFn: ({ tripId, input }) => financeApi.createExpense(tripId, input),
    ...options,
    onSuccess: (data, vars, ...rest) => {
      invalidateTripFinance(qc, vars.tripId);
      return options?.onSuccess?.(data, vars, ...rest);
    },
  });
}

type SetSharePaidVars = {
  tripId: string;
  expenseId: string;
  userId: string;
  input: SetSharePaidInput;
};

export function useSetSharePaid(options?: MutOpts<Expense, SetSharePaidVars>) {
  const qc = useQueryClient();
  return useMutation<
    Expense,
    ApiError,
    SetSharePaidVars,
    { previous?: Expense[] }
  >({
    mutationFn: ({ expenseId, userId, input }) =>
      financeApi.setSharePaid(expenseId, userId, input),
    ...options,
    onMutate: async (vars) => {
      const key = financeKeys.expenses(vars.tripId);
      await qc.cancelQueries({ queryKey: key });
      const previous = qc.getQueryData<Expense[]>(key);
      if (previous) {
        const nowIso = new Date().toISOString();
        qc.setQueryData<Expense[]>(
          key,
          previous.map((e) =>
            e.id === vars.expenseId
              ? {
                  ...e,
                  shares: e.shares.map((s) =>
                    s.user_id === vars.userId
                      ? {
                          ...s,
                          paid_at: vars.input.paid ? nowIso : undefined,
                        }
                      : s,
                  ),
                }
              : e,
          ),
        );
      }
      return { previous };
    },
    onError: (_err, vars, ctx) => {
      if (ctx?.previous) {
        qc.setQueryData(financeKeys.expenses(vars.tripId), ctx.previous);
      }
    },
    onSuccess: (data, vars, ...rest) => {
      void qc.invalidateQueries({
        queryKey: financeKeys.expenses(vars.tripId),
      });
      void qc.invalidateQueries({
        queryKey: financeKeys.personalStats(vars.tripId),
      });
      return options?.onSuccess?.(data, vars, ...rest);
    },
  });
}

type DeleteExpenseVars = { tripId: string; expenseId: string };

export function useDeleteExpense(options?: MutOpts<null, DeleteExpenseVars>) {
  const qc = useQueryClient();
  return useMutation<null, ApiError, DeleteExpenseVars>({
    mutationFn: ({ expenseId }) => financeApi.deleteExpense(expenseId),
    ...options,
    onSuccess: (data, vars, ...rest) => {
      invalidateTripFinance(qc, vars.tripId);
      return options?.onSuccess?.(data, vars, ...rest);
    },
  });
}

type UpsertBudgetVars = { tripId: string; input: UpsertBudgetInput };

export function useUpsertBudget(options?: MutOpts<Budget, UpsertBudgetVars>) {
  const qc = useQueryClient();
  return useMutation<Budget, ApiError, UpsertBudgetVars>({
    mutationFn: ({ tripId, input }) => financeApi.upsertBudget(tripId, input),
    ...options,
    onSuccess: (data, vars, ...rest) => {
      void qc.invalidateQueries({
        queryKey: financeKeys.budgets(vars.tripId),
      });
      return options?.onSuccess?.(data, vars, ...rest);
    },
  });
}

type DeleteBudgetVars = { tripId: string; budgetId: string };

export function useDeleteBudget(options?: MutOpts<null, DeleteBudgetVars>) {
  const qc = useQueryClient();
  return useMutation<null, ApiError, DeleteBudgetVars>({
    mutationFn: ({ tripId, budgetId }) =>
      financeApi.deleteBudget(tripId, budgetId),
    ...options,
    onSuccess: (data, vars, ...rest) => {
      void qc.invalidateQueries({
        queryKey: financeKeys.budgets(vars.tripId),
      });
      return options?.onSuccess?.(data, vars, ...rest);
    },
  });
}
