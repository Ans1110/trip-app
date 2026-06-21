"use client";

import { useMemo, useState, type FormEvent, type KeyboardEvent } from "react";
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

import {
  errorMessage,
  useCreateTodo,
  useDeleteTodo,
  useReorderTodos,
  useTodos,
  useUpdateTodo,
} from "@/hooks/trip-hooks";
import type { Todo, TodoPriority } from "@/lib/trip-api";

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

  const [title, setTitle] = useState("");
  const [priority, setPriority] = useState<TodoPriority>("normal");
  const [tagDraft, setTagDraft] = useState("");
  const [tags, setTags] = useState<string[]>([]);
  const [filterTag, setFilterTag] = useState<string | null>(null);
  const [filterPriority, setFilterPriority] = useState<TodoPriority | null>(null);
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
    useSensor(KeyboardSensor, { coordinateGetter: sortableKeyboardCoordinates }),
  );

  const handleSubmit = (e: FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    const trimmed = title.trim();
    if (!trimmed) return;
    create.mutate(
      {
        id: tripId,
        input: {
          title: trimmed,
          priority,
          tags: tags.length ? tags : undefined,
        },
      },
      {
        onSuccess: () => {
          setTitle("");
          setPriority("normal");
          setTagDraft("");
          setTags([]);
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
    setTags([...tags, t]);
    setTagDraft("");
  };

  const handleTagKeyDown = (e: KeyboardEvent<HTMLInputElement>) => {
    if (e.key === "Enter" || e.key === ",") {
      e.preventDefault();
      handleAddTag();
    } else if (e.key === "Backspace" && !tagDraft && tags.length) {
      setTags(tags.slice(0, -1));
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
      <form
        onSubmit={handleSubmit}
        className="flex flex-col gap-2.5 p-4 rounded-xl"
        style={{ backgroundColor: "#121814", border: "1px solid #1F2A24" }}
      >
        <div className="flex items-center gap-2">
          <input
            value={title}
            onChange={(e) => setTitle(e.target.value)}
            placeholder="Add a task…"
            className={`flex-1 ${inputClass}`}
            style={inputStyle}
          />
          <PrioritySelect value={priority} onChange={setPriority} />
          <button
            type="submit"
            disabled={create.isPending || !title.trim()}
            className="season-transition inline-flex items-center gap-1.5 px-4 py-2 text-sm font-medium rounded-full disabled:opacity-60"
            style={{
              backgroundColor: "var(--season-button)",
              color: "#0B100D",
            }}
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
            <TagChip
              key={t}
              tag={t}
              onRemove={() => setTags(tags.filter((x) => x !== t))}
            />
          ))}
          <div className="flex items-center gap-1">
            <Tag className="size-3" style={{ color: "#8B9A8E" }} />
            <input
              value={tagDraft}
              onChange={(e) => setTagDraft(e.target.value)}
              onKeyDown={handleTagKeyDown}
              onBlur={handleAddTag}
              placeholder="add tag"
              className="text-xs bg-transparent outline-none"
              style={{ color: "#ECEFEA", width: 90 }}
            />
          </div>
        </div>
        {create.isError && (
          <p className="text-xs" style={{ color: "#FCA5A5" }}>
            {errorMessage(create.error)}
          </p>
        )}
      </form>

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

function PrioritySelect({
  value,
  onChange,
}: {
  value: TodoPriority;
  onChange: (p: TodoPriority) => void;
}) {
  return (
    <select
      value={value}
      onChange={(e) => onChange(e.target.value as TodoPriority)}
      className="text-sm px-2 py-2 rounded-lg outline-none"
      style={inputStyle}
      aria-label="Priority"
    >
      {PRIORITIES.map((p) => (
        <option key={p} value={p}>
          {p}
        </option>
      ))}
    </select>
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
          ? accent?.bg ?? "rgba(181,208,134,0.18)"
          : "transparent",
        color: active ? accent?.fg ?? "#B5D086" : "#8B9A8E",
        border: `1px solid ${active ? accent?.fg ?? "#B5D086" : "#1F2A24"}`,
      }}
    >
      {label}
    </button>
  );
}

function SortableTodoRow({ tripId, todo }: { tripId: string; todo: Todo }) {
  const { attributes, listeners, setNodeRef, transform, transition, isDragging } =
    useSortable({ id: todo.id });
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
      <input
        type="checkbox"
        checked={todo.is_completed}
        onChange={(e) =>
          update.mutate({
            id: tripId,
            todoId: todo.id,
            input: { is_completed: e.target.checked },
          })
        }
        className="size-4 rounded accent-[color:var(--season-button)]"
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
      <select
        value={todo.priority}
        onChange={(e) =>
          update.mutate({
            id: tripId,
            todoId: todo.id,
            input: { priority: e.target.value as TodoPriority },
          })
        }
        aria-label="Change priority"
        className="text-xs px-2 py-1 rounded-full outline-none"
        style={{
          backgroundColor: accent.bg,
          color: accent.fg,
          border: `1px solid ${accent.fg}33`,
        }}
      >
        {PRIORITIES.map((p) => (
          <option
            key={p}
            value={p}
            style={{ backgroundColor: "#121814", color: "#ECEFEA" }}
          >
            {p}
          </option>
        ))}
      </select>
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

const inputClass =
  "px-3 py-2 rounded-lg text-sm outline-none focus:border-[color:var(--season-button)]";
const inputStyle: React.CSSProperties = {
  backgroundColor: "#161E19",
  border: "1px solid #1F2A24",
  color: "#ECEFEA",
};
