"use client";

import { useMemo, useState } from "react";
import { Controller, FormProvider, useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import {
  Check,
  Download,
  Loader2,
  Plus,
  Receipt,
  Trash2,
  Wallet,
} from "lucide-react";

import { Checkbox } from "@/components/ui/checkbox";
import { ControlledInput } from "@/components/ui/controlled-input";
import { ControlledDatePicker } from "@/components/ui/controlled-date-picker";
import { ControlledTimePicker } from "@/components/ui/controlled-time-picker";
import { useMe } from "@/hooks/auth-hooks";
import { useRoom } from "@/hooks/trip-hooks";
import {
  errorMessage,
  useBudgets,
  useCreateExpense,
  useDeleteBudget,
  useDeleteExpense,
  useExpenses,
  usePersonalStats,
  useSetSharePaid,
  useUpsertBudget,
} from "@/hooks/finance-hooks";
import {
  financeApi,
  type Budget,
  type CreateExpenseInput,
  type Expense,
} from "@/lib/apis/finance-api";
import type { UserSummary } from "@/lib/apis/trip-api";
import {
  createExpenseSchema,
  upsertBudgetSchema,
  type CreateExpenseFormInput,
  type UpsertBudgetFormInput,
} from "@/lib/schemas/finance-schemas";

import { Modal } from "./modal";

type MemberMap = Map<string, UserSummary>;

export function TripFinance({ tripId }: { tripId: string }) {
  const meQuery = useMe();
  const roomQuery = useRoom(tripId);
  const expenses = useExpenses(tripId);
  const budgets = useBudgets(tripId);
  const stats = usePersonalStats(tripId);

  const memberById = useMemo<MemberMap>(() => {
    const map = new Map<string, UserSummary>();
    for (const m of roomQuery.data?.members ?? []) {
      map.set(m.user.id, m.user);
    }
    return map;
  }, [roomQuery.data]);

  const memberList = useMemo(
    () => (roomQuery.data?.members ?? []).map((m) => m.user),
    [roomQuery.data],
  );

  // Base currency lives on the trip but isn't exposed in Trip payloads yet, so
  // stats is the authoritative source. Everything else assumes this single
  // implied unit — no per-expense currency picker, no FX rendering.
  const baseCurrency = stats.data?.base_currency ?? "TWD";
  const meId = meQuery.data?.id;

  return (
    <section className="flex flex-col gap-6">
      <HeroCard
        tripId={tripId}
        netBalance={stats.data?.net_balance}
        currency={baseCurrency}
        loading={stats.isLoading}
        error={errorMessage(stats.error)}
      />

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
        <ExpensesPanel
          tripId={tripId}
          meId={meId}
          memberById={memberById}
          memberList={memberList}
          currency={baseCurrency}
          loading={expenses.isLoading}
          error={errorMessage(expenses.error)}
          expenses={expenses.data ?? []}
        />
        <BudgetsPanel
          tripId={tripId}
          currency={baseCurrency}
          loading={budgets.isLoading}
          error={errorMessage(budgets.error)}
          budgets={budgets.data ?? []}
        />
      </div>
    </section>
  );
}

// ---- Hero ------------------------------------------------------------------

function HeroCard({
  tripId,
  netBalance,
  currency,
  loading,
  error,
}: {
  tripId: string;
  netBalance?: string;
  currency: string;
  loading: boolean;
  error: string | null;
}) {
  const net = toNumber(netBalance);
  const positive = net > 0;
  const negative = net < 0;
  const tone = positive ? "#B5D086" : negative ? "#FCA5A5" : "#8B9A8E";
  const label = positive
    ? "Need to charge"
    : negative
      ? "You owe"
      : "All settled";

  return (
    <div
      className="relative overflow-hidden rounded-2xl p-5"
      style={{
        backgroundColor: "#0F1512",
        border: "1px solid #1F2A24",
        backgroundImage:
          "radial-gradient(circle at top right, color-mix(in srgb, var(--season-button) 14%, transparent), transparent 55%)",
      }}
    >
      <div className="flex items-start justify-between gap-4 flex-wrap">
        <div className="flex flex-col gap-1 min-w-0">
          <span
            className="text-[11px] uppercase tracking-widest"
            style={{ color: "#6B7A6F" }}
          >
            {label} · {currency}
          </span>
          {loading ? (
            <span className="text-3xl text-[#ECEFEA]">…</span>
          ) : (
            <span
              className="text-3xl font-semibold tabular-nums"
              style={{
                color: tone,
                fontFamily: "var(--font-display, Georgia, serif)",
              }}
            >
              {formatAmount(Math.abs(net))}
            </span>
          )}
        </div>
        <a
          href={financeApi.exportCsvUrl(tripId, {})}
          className="inline-flex items-center gap-1.5 px-3 py-1.5 text-xs rounded-full hover:bg-white/5 shrink-0"
          style={{ color: "#ECEFEA", border: "1px solid #1F2A24" }}
        >
          <Download className="size-3.5" />
          Export CSV
        </a>
      </div>

      {error && <p className="text-xs text-[#FCA5A5] mt-3">{error}</p>}
    </div>
  );
}

// ---- Expenses --------------------------------------------------------------

function ExpensesPanel({
  tripId,
  meId,
  memberById,
  memberList,
  currency,
  loading,
  error,
  expenses,
}: {
  tripId: string;
  meId: string | undefined;
  memberById: MemberMap;
  memberList: UserSummary[];
  currency: string;
  loading: boolean;
  error: string | null;
  expenses: Expense[];
}) {
  const [adding, setAdding] = useState(false);
  return (
    <Panel
      icon={<Receipt className="size-3.5" />}
      title="Expenses"
      hint={countLabel(expenses.length, "expense")}
      action={
        <PanelButton
          onClick={() => setAdding(true)}
          disabled={memberList.length === 0}
          primary
        >
          <Plus className="size-3.5" /> Add
        </PanelButton>
      }
    >
      {loading && <PanelLoading />}
      {error && <PanelError message={error} />}
      {!loading && !error && expenses.length === 0 && (
        <PanelEmpty
          icon={<Receipt className="size-4" />}
          label="No expenses yet"
        />
      )}
      <ul className="flex flex-col divide-y" style={{ borderColor: "#1F2A24" }}>
        {expenses.map((e) => (
          <ExpenseRow
            key={e.id}
            tripId={tripId}
            expense={e}
            meId={meId}
            memberById={memberById}
          />
        ))}
      </ul>

      {adding && (
        <AddExpenseDialog
          tripId={tripId}
          meId={meId}
          memberList={memberList}
          currency={currency}
          onClose={() => setAdding(false)}
        />
      )}
    </Panel>
  );
}

function ExpenseRow({
  tripId,
  expense,
  meId,
  memberById,
}: {
  tripId: string;
  expense: Expense;
  meId: string | undefined;
  memberById: MemberMap;
}) {
  const del = useDeleteExpense();
  const setPaid = useSetSharePaid();
  const isCreator = meId != null && meId === expense.created_by;
  const payerName = memberById.get(expense.paid_by)?.name ?? "Someone";
  // Fully settled = every non-payer participant has a paid_at. The payer is
  // implicit (they can't owe themselves), so we only count the others.
  const others = expense.shares.filter((s) => s.user_id !== expense.paid_by);
  const settled = others.length > 0 && others.every((s) => !!s.paid_at);

  const handleDelete = () => {
    if (!window.confirm("Delete this expense?")) return;
    del.mutate({ tripId, expenseId: expense.id });
  };

  const handleToggle = (userId: string, nextPaid: boolean) => {
    setPaid.mutate({
      tripId,
      expenseId: expense.id,
      userId,
      input: { paid: nextPaid },
    });
  };

  return (
    <li
      className="flex flex-col gap-2 py-2.5 season-transition"
      style={{ opacity: settled ? 0.55 : 1 }}
    >
      <div className="flex items-center gap-3">
        <div className="flex flex-col min-w-0 flex-1">
          <div className="flex items-center gap-2 min-w-0">
            <span
              className="text-sm text-[#ECEFEA] truncate"
              style={{ textDecoration: settled ? "line-through" : "none" }}
            >
              {expense.description || "Untitled expense"}
            </span>
            {expense.category && <CategoryTag label={expense.category} />}
            {settled && <SettledChip />}
          </div>
          <span className="text-[11px]" style={{ color: "#6B7A6F" }}>
            {payerName} paid · {formatDate(expense.occurred_at)}
          </span>
        </div>
        <span className="text-sm tabular-nums text-[#ECEFEA] shrink-0">
          {formatAmount(toNumber(expense.amount))}
        </span>
        {isCreator && (
          <IconButton
            aria-label="Delete expense"
            onClick={handleDelete}
            busy={del.isPending}
            tone="danger"
          >
            <Trash2 className="size-3.5" />
          </IconButton>
        )}
      </div>
      <ParticipantAvatars
        shares={expense.shares}
        payerId={expense.paid_by}
        memberById={memberById}
        canToggle={isCreator}
        onToggle={handleToggle}
      />
    </li>
  );
}

function SettledChip() {
  return (
    <span
      className="inline-flex items-center gap-1 text-[10px] px-1.5 py-0.5 rounded-full shrink-0"
      style={{
        backgroundColor:
          "color-mix(in srgb, var(--season-button) 20%, transparent)",
        color: "var(--season-button)",
      }}
    >
      <Check className="size-3" />
      Settled
    </span>
  );
}

function ParticipantAvatars({
  shares,
  payerId,
  memberById,
  canToggle,
  onToggle,
}: {
  shares: Expense["shares"];
  payerId: string;
  memberById: MemberMap;
  canToggle: boolean;
  onToggle: (userId: string, nextPaid: boolean) => void;
}) {
  if (shares.length === 0) return null;
  return (
    <div className="flex flex-wrap items-center gap-1.5 pl-0.5">
      {shares.map((s) => {
        const user = memberById.get(s.user_id);
        const name = user?.name ?? "?";
        const isPayer = s.user_id === payerId;
        // The payer's own share is implicitly settled — they fronted the cash
        // so they can't owe themselves. Show it as "paid" without a toggle.
        const paid = isPayer || !!s.paid_at;
        return (
          <Avatar
            key={s.user_id}
            label={initials(name)}
            avatarUrl={user?.avatar_url}
            title={`${name}${isPayer ? " (payer)" : paid ? " · paid" : ""}`}
            paid={paid}
            isPayer={isPayer}
            interactive={canToggle && !isPayer}
            onClick={() => onToggle(s.user_id, !paid)}
          />
        );
      })}
    </div>
  );
}

function Avatar({
  label,
  avatarUrl,
  title,
  paid,
  isPayer,
  interactive,
  onClick,
}: {
  label: string;
  avatarUrl?: string;
  title: string;
  paid: boolean;
  isPayer: boolean;
  interactive: boolean;
  onClick: () => void;
}) {
  const common =
    "inline-flex items-center justify-center size-7 rounded-full text-[10px] font-medium select-none season-transition overflow-hidden";
  const style: React.CSSProperties = {
    backgroundColor: paid
      ? "#1F2A24"
      : "color-mix(in srgb, var(--season-button) 24%, transparent)",
    color: paid ? "#6B7A6F" : "#ECEFEA",
    border: isPayer
      ? "1px solid var(--season-button)"
      : "1px solid transparent",
    opacity: paid ? 0.55 : 1,
    textDecoration: paid && !isPayer ? "line-through" : "none",
  };
  const content = avatarUrl ? (
    // eslint-disable-next-line @next/next/no-img-element
    <img src={avatarUrl} alt="" className="size-full object-cover" />
  ) : (
    label
  );
  if (!interactive) {
    return (
      <span className={common} title={title} style={style}>
        {content}
      </span>
    );
  }
  return (
    <button
      type="button"
      onClick={onClick}
      title={title}
      aria-pressed={paid}
      className={`${common} hover:brightness-110`}
      style={style}
    >
      {content}
    </button>
  );
}

function CategoryTag({ label }: { label: string }) {
  return (
    <span
      className="text-[10px] px-1.5 py-0.5 rounded-full shrink-0"
      style={{
        backgroundColor:
          "color-mix(in srgb, var(--season-button) 14%, transparent)",
        color: "#ECEFEA",
      }}
    >
      {label}
    </span>
  );
}

function AddExpenseDialog({
  tripId,
  meId,
  memberList,
  currency,
  onClose,
}: {
  tripId: string;
  meId: string | undefined;
  memberList: UserSummary[];
  currency: string;
  onClose: () => void;
}) {
  const create = useCreateExpense();
  const now = new Date();

  const form = useForm<CreateExpenseFormInput>({
    resolver: zodResolver(createExpenseSchema),
    defaultValues: {
      amount: "",
      description: "",
      category: "",
      occurred_date: toDateInputValue(now),
      occurred_time: toTimeInputValue(now),
      participants: memberList.map((m) => m.id),
    },
  });
  const { handleSubmit } = form;

  const onSubmit = (v: CreateExpenseFormInput) => {
    if (!meId) return;
    const participants = v.participants.includes(meId)
      ? v.participants
      : [meId, ...v.participants];
    const input: CreateExpenseInput = {
      paid_by: meId,
      amount: v.amount,
      currency,
      description: v.description.trim() || undefined,
      category: v.category.trim() || undefined,
      split_strategy: "equal",
      participants,
      occurred_at: new Date(
        `${v.occurred_date}T${v.occurred_time}`,
      ).toISOString(),
    };
    create.mutate({ tripId, input }, { onSuccess: () => onClose() });
  };

  const submitError = errorMessage(create.error);

  return (
    <Modal title="Add expense" onClose={onClose}>
      <FormProvider {...form}>
        <form
          className="flex flex-col gap-4"
          onSubmit={handleSubmit(onSubmit)}
          noValidate
        >
          <ControlledInput<CreateExpenseFormInput>
            name="description"
            label="Description"
            placeholder="Dinner at izakaya"
          />

          <div className="grid grid-cols-2 gap-3">
            <ControlledInput<CreateExpenseFormInput>
              name="amount"
              label={`Amount (${currency})`}
              inputMode="decimal"
              placeholder="0.00"
            />
            <ControlledInput<CreateExpenseFormInput>
              name="category"
              label="Category"
              placeholder="Food"
            />
          </div>

          <div className="grid grid-cols-2 gap-3">
            <ControlledDatePicker<CreateExpenseFormInput>
              name="occurred_date"
              placeholder="Date"
            />
            <ControlledTimePicker<CreateExpenseFormInput>
              name="occurred_time"
              placeholder="Time"
            />
          </div>

          <ParticipantsField meId={meId} memberList={memberList} />

          {submitError && (
            <p className="text-sm text-[#FCA5A5]">{submitError}</p>
          )}

          <DialogFooter
            onCancel={onClose}
            submitLabel="Save expense"
            pending={create.isPending}
          />
        </form>
      </FormProvider>
    </Modal>
  );
}

function ParticipantsField({
  meId,
  memberList,
}: {
  meId: string | undefined;
  memberList: UserSummary[];
}) {
  return (
    <Controller<CreateExpenseFormInput, "participants">
      name="participants"
      render={({ field, fieldState: { error } }) => {
        const selected = new Set(field.value ?? []);
        const toggle = (id: string) => {
          const next = new Set(selected);
          if (next.has(id)) next.delete(id);
          else next.add(id);
          field.onChange(Array.from(next));
        };
        return (
          <div className="flex flex-col gap-1.5">
            <span
              className="text-[11px] uppercase tracking-widest"
              style={{ color: "#6B7A6F" }}
            >
              Split between
            </span>
            <div
              className="flex flex-wrap gap-1.5 p-2 rounded-lg"
              style={{
                backgroundColor: "#0F1512",
                border: "1px solid #1F2A24",
              }}
            >
              {memberList.map((m) => {
                const isMe = m.id === meId;
                const checked = isMe || selected.has(m.id);
                return (
                  <label
                    key={m.id}
                    className="inline-flex items-center gap-2 px-2 py-1 rounded-full text-xs cursor-pointer hover:bg-white/5"
                    style={{ color: "#ECEFEA" }}
                  >
                    <Checkbox
                      checked={checked}
                      disabled={isMe}
                      onCheckedChange={() => !isMe && toggle(m.id)}
                    />
                    <span>
                      {m.name}
                      {isMe ? " (you)" : ""}
                    </span>
                  </label>
                );
              })}
            </div>
            {error && (
              <span className="text-[11px] text-[#FCA5A5]">
                {error.message}
              </span>
            )}
          </div>
        );
      }}
    />
  );
}

// ---- Budgets ---------------------------------------------------------------

function BudgetsPanel({
  tripId,
  currency,
  loading,
  error,
  budgets,
}: {
  tripId: string;
  currency: string;
  loading: boolean;
  error: string | null;
  budgets: Budget[];
}) {
  const [editing, setEditing] = useState(false);
  const total = useMemo(
    () => budgets.reduce((sum, b) => sum + toNumber(b.amount), 0),
    [budgets],
  );
  return (
    <Panel
      icon={<Wallet className="size-3.5" />}
      title="Budgets"
      hint={countLabel(budgets.length, "budget")}
      action={
        <PanelButton onClick={() => setEditing(true)} primary>
          <Plus className="size-3.5" /> Add
        </PanelButton>
      }
    >
      {loading && <PanelLoading />}
      {error && <PanelError message={error} />}
      {!loading && !error && budgets.length === 0 && (
        <PanelEmpty
          icon={<Wallet className="size-4" />}
          label="No budgets yet"
        />
      )}
      <ul className="flex flex-col divide-y" style={{ borderColor: "#1F2A24" }}>
        {budgets.map((b) => (
          <BudgetRow key={b.id} tripId={tripId} budget={b} />
        ))}
      </ul>
      {budgets.length > 0 && (
        <div
          className="flex items-center justify-between pt-2 mt-1 border-t"
          style={{ borderColor: "#1F2A24" }}
        >
          <span
            className="text-[11px] uppercase tracking-widest"
            style={{ color: "#6B7A6F" }}
          >
            Total · {currency}
          </span>
          <span className="text-sm font-medium tabular-nums text-[#ECEFEA]">
            {formatAmount(total)}
          </span>
        </div>
      )}

      {editing && (
        <UpsertBudgetDialog
          tripId={tripId}
          currency={currency}
          onClose={() => setEditing(false)}
        />
      )}
    </Panel>
  );
}

function BudgetRow({ tripId, budget }: { tripId: string; budget: Budget }) {
  const del = useDeleteBudget();
  const total = toNumber(budget.amount);

  const handleDelete = () => {
    if (!window.confirm(`Delete the ${budget.category} budget?`)) return;
    del.mutate({ tripId, budgetId: budget.id });
  };

  return (
    <li className="flex items-center justify-between gap-2 py-2.5">
      <span className="text-sm text-[#ECEFEA] truncate">{budget.category}</span>
      <div className="flex items-center gap-2 shrink-0">
        <span className="text-sm tabular-nums text-[#ECEFEA]">
          {formatAmount(total)}
        </span>
        <IconButton
          aria-label="Delete budget"
          onClick={handleDelete}
          busy={del.isPending}
          tone="danger"
        >
          <Trash2 className="size-3.5" />
        </IconButton>
      </div>
    </li>
  );
}

function UpsertBudgetDialog({
  tripId,
  currency,
  onClose,
}: {
  tripId: string;
  currency: string;
  onClose: () => void;
}) {
  const upsert = useUpsertBudget();
  const form = useForm<UpsertBudgetFormInput>({
    resolver: zodResolver(upsertBudgetSchema),
    defaultValues: { category: "", amount: "" },
  });
  const { handleSubmit } = form;

  const onSubmit = (v: UpsertBudgetFormInput) => {
    upsert.mutate(
      {
        tripId,
        input: {
          category: v.category.trim(),
          amount: v.amount,
          currency,
        },
      },
      { onSuccess: () => onClose() },
    );
  };

  const submitError = errorMessage(upsert.error);
  return (
    <Modal title="Set budget" onClose={onClose}>
      <FormProvider {...form}>
        <form
          className="flex flex-col gap-4"
          onSubmit={handleSubmit(onSubmit)}
          noValidate
        >
          <ControlledInput<UpsertBudgetFormInput>
            name="category"
            label="Category"
            placeholder="Food, transport, lodging…"
          />
          <ControlledInput<UpsertBudgetFormInput>
            name="amount"
            label={`Cap (${currency})`}
            inputMode="decimal"
            placeholder="0.00"
          />
          {submitError && (
            <p className="text-sm text-[#FCA5A5]">{submitError}</p>
          )}
          <DialogFooter
            onCancel={onClose}
            submitLabel="Save budget"
            pending={upsert.isPending}
          />
        </form>
      </FormProvider>
    </Modal>
  );
}

// ---- Shared UI atoms -------------------------------------------------------

function Panel({
  icon,
  title,
  hint,
  action,
  children,
}: {
  icon: React.ReactNode;
  title: string;
  hint: string;
  action?: React.ReactNode;
  children: React.ReactNode;
}) {
  return (
    <section
      className="flex flex-col gap-3 rounded-2xl p-4"
      style={{ backgroundColor: "#121814", border: "1px solid #1F2A24" }}
    >
      <div className="flex items-center justify-between gap-3">
        <div className="flex items-center gap-2 min-w-0">
          <span
            className="inline-flex items-center justify-center size-6 rounded-md"
            style={{
              backgroundColor:
                "color-mix(in srgb, var(--season-button) 16%, transparent)",
              color: "#ECEFEA",
            }}
          >
            {icon}
          </span>
          <div className="flex flex-col min-w-0">
            <span className="text-sm font-medium text-[#ECEFEA]">{title}</span>
            <span className="text-[11px]" style={{ color: "#6B7A6F" }}>
              {hint}
            </span>
          </div>
        </div>
        {action}
      </div>
      {children}
    </section>
  );
}

function PanelButton({
  children,
  onClick,
  disabled,
  primary,
}: {
  children: React.ReactNode;
  onClick: () => void;
  disabled?: boolean;
  primary?: boolean;
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      disabled={disabled}
      className="season-transition inline-flex items-center gap-1 px-3 py-1.5 text-xs font-medium rounded-full hover:bg-white/5 disabled:opacity-60"
      style={
        primary
          ? {
              backgroundColor: "var(--season-button)",
              color: "#0B100D",
            }
          : { color: "#ECEFEA", border: "1px solid #1F2A24" }
      }
    >
      {children}
    </button>
  );
}

function PanelLoading() {
  return (
    <p className="text-xs" style={{ color: "#8B9A8E" }}>
      Loading…
    </p>
  );
}

function PanelError({ message }: { message: string }) {
  return <p className="text-xs text-[#FCA5A5]">{message}</p>;
}

function PanelEmpty({ icon, label }: { icon: React.ReactNode; label: string }) {
  return (
    <div
      className="flex items-center gap-2 px-3 py-3 rounded-lg text-xs"
      style={{
        backgroundColor: "#0F1512",
        border: "1px dashed #1F2A24",
        color: "#6B7A6F",
      }}
    >
      {icon}
      {label}
    </div>
  );
}

function IconButton({
  children,
  onClick,
  busy,
  tone,
  "aria-label": ariaLabel,
}: {
  children: React.ReactNode;
  onClick: () => void;
  busy?: boolean;
  tone: "ok" | "danger";
  "aria-label": string;
}) {
  const color = tone === "ok" ? "#B5D086" : "#FCA5A5";
  return (
    <button
      type="button"
      onClick={onClick}
      aria-label={ariaLabel}
      disabled={busy}
      className="inline-flex items-center justify-center size-7 rounded-full hover:bg-white/5 disabled:opacity-60 shrink-0"
      style={{ color }}
    >
      {busy ? <Loader2 className="size-3.5 animate-spin" /> : children}
    </button>
  );
}

function DialogFooter({
  onCancel,
  submitLabel,
  pending,
}: {
  onCancel: () => void;
  submitLabel: string;
  pending: boolean;
}) {
  return (
    <div className="flex items-center justify-end gap-2 pt-2">
      <button
        type="button"
        onClick={onCancel}
        disabled={pending}
        className="px-4 py-2 text-sm rounded-full hover:bg-white/5 disabled:opacity-60 text-[#8B9A8E]"
      >
        Cancel
      </button>
      <button
        type="submit"
        disabled={pending}
        className="season-transition inline-flex items-center gap-2 px-4 py-2 text-sm font-medium rounded-full disabled:opacity-60 text-[#0B100D]"
        style={{ backgroundColor: "var(--season-button)" }}
      >
        {pending && <Loader2 className="size-3.5 animate-spin" />}
        {submitLabel}
      </button>
    </div>
  );
}

// ---- Formatters ------------------------------------------------------------

// Amounts are already in the trip's base currency (see TripFinance). The
// currency label is rendered separately on the hero and dialogs, so numeric
// output is unit-less and tabular.
function formatAmount(n: number): string {
  if (!Number.isFinite(n)) return "0.00";
  return n.toLocaleString(undefined, {
    minimumFractionDigits: 2,
    maximumFractionDigits: 2,
  });
}

function toNumber(raw: string | undefined): number {
  if (!raw) return 0;
  const n = Number(raw);
  return Number.isFinite(n) ? n : 0;
}

function formatDate(iso: string): string {
  return new Date(iso).toLocaleDateString(undefined, {
    month: "short",
    day: "numeric",
  });
}

function toDateInputValue(d: Date): string {
  const y = d.getFullYear();
  const m = String(d.getMonth() + 1).padStart(2, "0");
  const day = String(d.getDate()).padStart(2, "0");
  return `${y}-${m}-${day}`;
}

function toTimeInputValue(d: Date): string {
  const h = String(d.getHours()).padStart(2, "0");
  const min = String(d.getMinutes()).padStart(2, "0");
  return `${h}:${min}`;
}

function countLabel(n: number, unit: string): string {
  if (n === 0) return `No ${unit}s`;
  return `${n} ${unit}${n === 1 ? "" : "s"}`;
}

// Placeholder avatar label: pull the first Latin/CJK grapheme (best-effort) so
// names like "Alice Wang" render as "A" and "王小明" renders as "王".
function initials(name: string): string {
  const trimmed = name.trim();
  if (!trimmed) return "?";
  const chars = Array.from(trimmed);
  return (chars[0] ?? "?").toUpperCase();
}
