"use client";

import { useState, type FormEvent } from "react";
import { Loader2, Plus, Trash2 } from "lucide-react";

import {
  errorMessage,
  useCreateTodo,
  useDeleteTodo,
  useTodos,
  useUpdateTodo,
} from "@/hooks/trip-hooks";
import type { Todo } from "@/lib/trip-api";

export function TripTodos({ tripId }: { tripId: string }) {
  const query = useTodos(tripId);
  const create = useCreateTodo();
  const [title, setTitle] = useState("");

  const todos = query.data ?? [];

  const handleSubmit = (e: FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    const trimmed = title.trim();
    if (!trimmed) return;
    create.mutate(
      { id: tripId, input: { title: trimmed } },
      { onSuccess: () => setTitle("") },
    );
  };

  return (
    <section className="flex flex-col gap-4">
      <form onSubmit={handleSubmit} className="flex items-center gap-2">
        <input
          value={title}
          onChange={(e) => setTitle(e.target.value)}
          placeholder="Add a task…"
          className="flex-1 px-3 py-2 rounded-lg text-sm outline-none focus:border-[color:var(--season-button)]"
          style={{
            backgroundColor: "#161E19",
            border: "1px solid #1F2A24",
            color: "#ECEFEA",
          }}
        />
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
      </form>
      {create.isError && (
        <p className="text-xs" style={{ color: "#FCA5A5" }}>
          {errorMessage(create.error)}
        </p>
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
      {!query.isLoading && !query.isError && todos.length === 0 && (
        <p className="text-sm" style={{ color: "#8B9A8E" }}>
          Nothing on the list yet.
        </p>
      )}

      <div className="flex flex-col gap-2">
        {todos.map((todo) => (
          <TodoRow key={todo.id} tripId={tripId} todo={todo} />
        ))}
      </div>
    </section>
  );
}

function TodoRow({ tripId, todo }: { tripId: string; todo: Todo }) {
  const update = useUpdateTodo();
  const remove = useDeleteTodo();

  return (
    <div
      className="flex items-center gap-3 px-4 py-3 rounded-lg"
      style={{ backgroundColor: "#121814", border: "1px solid #1F2A24" }}
    >
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
      <p
        className={`flex-1 text-sm ${todo.is_completed ? "line-through" : ""}`}
        style={{ color: todo.is_completed ? "#6B7A6F" : "#ECEFEA" }}
      >
        {todo.title}
      </p>
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
