"use client";

import { useMemo, useState, type KeyboardEvent } from "react";
import { FormProvider, useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { GripVertical, Loader2, Plus, Tag, Trash2, X } from "lucide-react";
import {
  DndContext,
  KeyboardSensor,
  PointerSensor,
  closestCenter,
  useSensor,
  useSensors,
  type DragEndEvent,
} from "@dnd-kit/core";
import {
  SortableContext,
  arrayMove,
  sortableKeyboardCoordinates,
  useSortable,
  verticalListSortingStrategy,
} from "@dnd-kit/sortable";
import { CSS } from "@dnd-kit/utilities";

import { Checkbox } from "@/components/ui/checkbox";
import { ControlledInput } from "@/components/ui/controlled-input";
import { ControlledSelect } from "@/components/ui/controlled-select";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import {
  errorMessage,
  useCreateTodo,
  useDeleteTodo,
  useReorderTodos,
  useTodos,
  useUpdateTodo,
} from "@/hooks/trip-hooks";
import type { Todo, TodoPriority } from "@/lib/apis/trip-api";
import {
  createTodoSchema,
  type CreateTodoFormInput,
} from "@/lib/schemas/trip-schemas";

const PRIORITIES: TodoPriority[] = ["low", "normal", "high"];

const priorityColor: Record<TodoPriority, { bg: string; fg: string }> = {
  low: { bg: "rgba(139,154,142,0.18)", fg: "#9CB0A3" },
  normal: { bg: "rgba(181,208,134,0.18)", fg: "#B5D086" },
  high: { bg: "rgba(252,165,165,0.18)", fg: "#FCA5A5" },
};

const normalizeTag = (raw: string) => raw.trim().toLowerCase();

export function TripTodos({ tripId }: { tripId: string }) {
  const query = useTodos(tripId);
  const create = useCreateTodo();
  const reorder = useReorderTodos();

  const form = useForm<CreateTodoFormInput>({
    resolver: zodResolver(createTodoSchema),
    defaultValues: { title: "", priority: "normal", tags: [] },
  });
  const {
    handleSubmit,
    reset,
    watch,
    setValue,
    formState: { errors },
  } = form;
  const tags = watch("tags");
  const titleValue = watch("title");

  const [tagDraft, setTagDraft] = useState("");
  const [filterTag, setFilterTag] = useState<string | null>(null);
  const [filterPriority, setFilterPriority] = useState<TodoPriority | null>(
    null,
  );
  const [orderOverride, setOrderOverride] = useState<Todo[] | null>(null);

  const todos = query.data ?? [];
  const sorted = useMemo(
    () => [...todos].sort((a, b) => a.sort_order - b.sort_order),
    [todos],
  );
  const display = orderOverride ?? sorted;

  const tagOptions = useMemo(() => {
    const set = new Set<string>();
    for (const t of todos) for (const tag of t.tags) set.add(tag);
    return [...set].sort();
  }, [todos]);

  const visible = useMemo(
    () =>
      display.filter((t) => {
        if (filterTag && !t.tags.includes(filterTag)) return false;
        if (filterPriority && t.priority !== filterPriority) return false;
        return true;
      }),
    [display, filterTag, filterPriority],
  );

  const sensors = useSensors(
    useSensor(PointerSensor, { activationConstraint: { distance: 6 } }),
    useSensor(KeyboardSensor, {
      coordinateGetter: sortableKeyboardCoordinates,
    }),
  );

  const onSubmit = (v: CreateTodoFormInput) => {
    create.mutate(
      {
        id: tripId,
        input: {
          title: v.title,
          priority: v.priority,
          tags: v.tags.length ? v.tags : undefined,
        },
      },
      {
        onSuccess: () => {
          reset({ title: "", priority: "normal", tags: [] });
          setTagDraft("");
        },
      },
    );
  };

  const handleAddTag = () => {
    const t = normalizeTag(tagDraft);
    if (!t) return;
    if (tags.includes(t)) {
      setTagDraft("");
      return;
    }
    setValue("tags", [...tags, t], { shouldValidate: true });
    setTagDraft("");
  };

  const handleRemoveTag = (tag: string) => {
    setValue(
      "tags",
      tags.filter((t) => t !== tag),
      { shouldValidate: true },
    );
  };

  const handleTagKeyDown = (e: KeyboardEvent<HTMLInputElement>) => {
    if (e.key === "Enter" || e.key === ",") {
      e.preventDefault();
      handleAddTag();
    } else if (e.key === "Backspace" && !tagDraft && tags.length) {
      setValue("tags", tags.slice(0, -1), { shouldValidate: true });
    }
  };

  const handleDragEnd = (e: DragEndEvent) => {
    const { active, over } = e;
    if (!over || active.id === over.id) return;
    const oldIndex = display.findIndex((t) => t.id === active.id);
    const newIndex = display.findIndex((t) => t.id === over.id);
    if (oldIndex < 0 || newIndex < 0) return;
    const next = arrayMove(display, oldIndex, newIndex);
    setOrderOverride(next);
    reorder.mutate(
      { id: tripId, input: { todo_ids: next.map((t) => t.id) } },
      { onSettled: () => setOrderOverride(null) },
    );
  };

  return (
    <section className="flex flex-col gap-4">
      <FormProvider {...form}>
        <form
          onSubmit={handleSubmit(onSubmit)}
          noValidate
          className="flex flex-col gap-2.5 p-4 rounded-xl bg-[#121814] border border-[#1F2A24]"
        >
          <div className="flex items-end gap-2">
            <ControlledInput<CreateTodoFormInput>
              name="title"
              placeholder="Add a task…"
              containerClassName="flex-1"
            />
            <ControlledSelect<CreateTodoFormInput>
              name="priority"
              aria-label="Priority"
              options={PRIORITIES.map((p) => ({ value: p, label: p }))}
              containerClassName="w-28"
            />
            <button
              type="submit"
              disabled={create.isPending || !titleValue.trim()}
              className="season-transition inline-flex items-center gap-1.5 px-4 py-2 text-sm font-medium rounded-full disabled:opacity-60 text-[#0B100D]"
              style={{ backgroundColor: "var(--season-button)" }}
            >
              {create.isPending ? (
                <Loader2 className="size-3.5 animate-spin" />
              ) : (
                <Plus className="size-4" />
              )}
              Add
            </button>
          </div>
          <div className="flex items-center gap-1.5 flex-wrap">
            {tags.map((t) => (
              <TagChip key={t} tag={t} onRemove={() => handleRemoveTag(t)} />
            ))}
            <div className="flex items-center gap-1">
              <Tag className="size-3 text-[#8B9A8E]" />
              <input
                value={tagDraft}
                onChange={(e) => setTagDraft(e.target.value)}
                onKeyDown={handleTagKeyDown}
                onBlur={handleAddTag}
                placeholder="add tag"
                className="text-xs bg-transparent outline-none text-[#ECEFEA]"
                style={{ width: 90 }}
              />
            </div>
          </div>
          {errors.tags && (
            <span className="text-[11px] text-[#FCA5A5]">
              {errors.tags.message}
            </span>
          )}
          {create.isError && (
            <p className="text-xs text-[#FCA5A5]">
              {errorMessage(create.error)}
            </p>
          )}
        </form>
      </FormProvider>

      {(tagOptions.length > 0 || todos.length > 0) && (
        <div className="flex items-center gap-1.5 flex-wrap">
          <span className="text-xs" style={{ color: "#6B7A6F" }}>
            Filter:
          </span>
          {PRIORITIES.map((p) => (
            <FilterChip
              key={p}
              label={p}
              active={filterPriority === p}
              onClick={() => setFilterPriority(filterPriority === p ? null : p)}
              accent={priorityColor[p]}
            />
          ))}
          {tagOptions.map((t) => (
            <FilterChip
              key={t}
              label={`#${t}`}
              active={filterTag === t}
              onClick={() => setFilterTag(filterTag === t ? null : t)}
            />
          ))}
          {(filterPriority || filterTag) && (
            <button
              type="button"
              onClick={() => {
                setFilterPriority(null);
                setFilterTag(null);
              }}
              className="text-xs px-2 py-1 rounded-full hover:bg-white/5"
              style={{ color: "#8B9A8E" }}
            >
              Clear
            </button>
          )}
        </div>
      )}

      {query.isLoading && (
        <p className="text-sm" style={{ color: "#8B9A8E" }}>
          Loading…
        </p>
      )}
      {query.isError && (
        <p className="text-sm" style={{ color: "#FCA5A5" }}>
          {errorMessage(query.error)}
        </p>
      )}
      {!query.isLoading && !query.isError && visible.length === 0 && (
        <p className="text-sm" style={{ color: "#8B9A8E" }}>
          {todos.length === 0
            ? "Nothing on the list yet."
            : "No tasks match the filters."}
        </p>
      )}

      {reorder.isPending && (
        <p className="text-xs" style={{ color: "#8B9A8E" }}>
          Reordering…
        </p>
      )}

      <DndContext
        sensors={sensors}
        collisionDetection={closestCenter}
        onDragEnd={handleDragEnd}
      >
        <SortableContext
          items={visible.map((t) => t.id)}
          strategy={verticalListSortingStrategy}
        >
          <div className="flex flex-col gap-2">
            {visible.map((todo) => (
              <SortableTodoRow key={todo.id} tripId={tripId} todo={todo} />
            ))}
          </div>
        </SortableContext>
      </DndContext>
    </section>
  );
}

function TagChip({ tag, onRemove }: { tag: string; onRemove: () => void }) {
  return (
    <span
      className="inline-flex items-center gap-1 text-xs px-2 py-0.5 rounded-full"
      style={{ backgroundColor: "#1F2A24", color: "#9CB0A3" }}
    >
      #{tag}
      <button
        type="button"
        onClick={onRemove}
        aria-label={`Remove ${tag}`}
        className="hover:text-white"
      >
        <X className="size-3" />
      </button>
    </span>
  );
}

function FilterChip({
  label,
  active,
  onClick,
  accent,
}: {
  label: string;
  active: boolean;
  onClick: () => void;
  accent?: { bg: string; fg: string };
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      className="text-xs px-2 py-1 rounded-full season-transition"
      style={{
        backgroundColor: active
          ? (accent?.bg ?? "rgba(181,208,134,0.18)")
          : "transparent",
        color: active ? (accent?.fg ?? "#B5D086") : "#8B9A8E",
        border: `1px solid ${active ? (accent?.fg ?? "#B5D086") : "#1F2A24"}`,
      }}
    >
      {label}
    </button>
  );
}

function SortableTodoRow({ tripId, todo }: { tripId: string; todo: Todo }) {
  const {
    attributes,
    listeners,
    setNodeRef,
    transform,
    transition,
    isDragging,
  } = useSortable({ id: todo.id });
  const style: React.CSSProperties = {
    transform: CSS.Transform.toString(transform),
    transition,
    opacity: isDragging ? 0.6 : 1,
  };
  return (
    <div ref={setNodeRef} style={style}>
      <TodoRow
        tripId={tripId}
        todo={todo}
        dragHandle={
          <button
            type="button"
            aria-label="Reorder"
            className="inline-flex items-center justify-center size-7 rounded-full hover:bg-white/5 cursor-grab active:cursor-grabbing touch-none"
            style={{ color: "#8B9A8E" }}
            {...attributes}
            {...listeners}
          >
            <GripVertical className="size-3.5" />
          </button>
        }
      />
    </div>
  );
}

function TodoRow({
  tripId,
  todo,
  dragHandle,
}: {
  tripId: string;
  todo: Todo;
  dragHandle?: React.ReactNode;
}) {
  const update = useUpdateTodo();
  const remove = useDeleteTodo();
  const accent = priorityColor[todo.priority];

  return (
    <div
      className="flex items-center gap-3 px-4 py-3 rounded-lg"
      style={{ backgroundColor: "#121814", border: "1px solid #1F2A24" }}
    >
      {dragHandle}
      <Checkbox
        checked={todo.is_completed}
        onCheckedChange={(v) =>
          update.mutate({
            id: tripId,
            todoId: todo.id,
            input: { is_completed: v === true },
          })
        }
        aria-label={todo.is_completed ? "Mark incomplete" : "Mark complete"}
      />
      <div className="flex-1 min-w-0 flex flex-col gap-1">
        <p
          className={`text-sm ${todo.is_completed ? "line-through" : ""}`}
          style={{ color: todo.is_completed ? "#6B7A6F" : "#ECEFEA" }}
        >
          {todo.title}
        </p>
        {todo.tags.length > 0 && (
          <div className="flex items-center gap-1 flex-wrap">
            {todo.tags.map((t) => (
              <span
                key={t}
                className="text-[10px] px-1.5 py-0.5 rounded-full"
                style={{ backgroundColor: "#1F2A24", color: "#9CB0A3" }}
              >
                #{t}
              </span>
            ))}
          </div>
        )}
      </div>
      <Select
        value={todo.priority}
        onValueChange={(v) =>
          update.mutate({
            id: tripId,
            todoId: todo.id,
            input: { priority: v as TodoPriority },
          })
        }
      >
        <SelectTrigger
          aria-label="Change priority"
          className="w-auto h-auto gap-1 text-xs px-2 py-1 rounded-full"
          style={{
            backgroundColor: accent.bg,
            color: accent.fg,
            borderColor: `${accent.fg}33`,
          }}
        >
          <SelectValue />
        </SelectTrigger>
        <SelectContent>
          {PRIORITIES.map((p) => (
            <SelectItem key={p} value={p}>
              {p}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>
      <button
        type="button"
        onClick={() => remove.mutate({ id: tripId, todoId: todo.id })}
        disabled={remove.isPending}
        aria-label="Delete"
        className="inline-flex items-center justify-center size-7 rounded-full hover:bg-white/5 disabled:opacity-60"
        style={{ color: "#FCA5A5" }}
      >
        {remove.isPending ? (
          <Loader2 className="size-3.5 animate-spin" />
        ) : (
          <Trash2 className="size-3.5" />
        )}
      </button>
    </div>
  );
}
