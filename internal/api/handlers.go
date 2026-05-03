package api

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/5afar/go_task_manager/internal/service"
	"github.com/5afar/go_task_manager/internal/store"
)

func RegisterRoutes(mux *http.ServeMux, svc *service.TaskService) {
	mux.HandleFunc("/tasks", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			listHandler(svc)(w, r)
		case http.MethodPost:
			createHandler(svc)(w, r)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/tasks/", func(w http.ResponseWriter, r *http.Request) {
		// path: /tasks/{id}
		id := strings.TrimPrefix(r.URL.Path, "/tasks/")
		if id == "" {
			http.NotFound(w, r)
			return
		}
		switch r.Method {
		case http.MethodGet:
			getHandler(svc)(w, r, id)
		case http.MethodPut:
			updateHandler(svc)(w, r, id)
		case http.MethodDelete:
			deleteHandler(svc)(w, r, id)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})
}

func writeJSON(w http.ResponseWriter, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": msg})
}

func createHandler(svc *service.TaskService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var t store.Task
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		if err := json.Unmarshal(body, &t); err != nil {
			http.Error(w, "invalid json", http.StatusBadRequest)
			return
		}
		if err := svc.CreateTask(context.Background(), &t); err != nil {
			if err == service.ErrInvalidTask {
				writeError(w, http.StatusBadRequest, "invalid task")
				return
			}
			writeError(w, http.StatusInternalServerError, "internal error")
			return
		}
		w.WriteHeader(http.StatusCreated)
		writeJSON(w, t)
	}
}

func listHandler(svc *service.TaskService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		limit := 20
		offset := 0
		if l := q.Get("limit"); l != "" {
			if v, err := strconv.Atoi(l); err == nil {
				limit = v
			}
		}
		if o := q.Get("offset"); o != "" {
			if v, err := strconv.Atoi(o); err == nil {
				offset = v
			}
		}
		res, err := svc.ListTasks(context.Background(), limit, offset)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "internal error")
			return
		}
		writeJSON(w, res)
	}
}

func getHandler(svc *service.TaskService) func(http.ResponseWriter, *http.Request, string) {
	return func(w http.ResponseWriter, r *http.Request, id string) {
		t, err := svc.GetTask(context.Background(), id)
		if err != nil {
			if err == service.ErrNotFound {
				writeError(w, http.StatusNotFound, "not found")
				return
			}
			writeError(w, http.StatusInternalServerError, "internal error")
			return
		}
		writeJSON(w, t)
	}
}

func updateHandler(svc *service.TaskService) func(http.ResponseWriter, *http.Request, string) {
	return func(w http.ResponseWriter, r *http.Request, id string) {
		var t store.Task
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		if err := json.Unmarshal(body, &t); err != nil {
			http.Error(w, "invalid json", http.StatusBadRequest)
			return
		}
		t.ID = id
		if err := svc.UpdateTask(context.Background(), &t); err != nil {
			if err == service.ErrInvalidTask {
				writeError(w, http.StatusBadRequest, "invalid task")
				return
			}
			writeError(w, http.StatusInternalServerError, "internal error")
			return
		}
		writeJSON(w, t)
	}
}

func deleteHandler(svc *service.TaskService) func(http.ResponseWriter, *http.Request, string) {
	return func(w http.ResponseWriter, r *http.Request, id string) {
		if err := svc.DeleteTask(context.Background(), id); err != nil {
			writeError(w, http.StatusInternalServerError, "internal error")
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}
