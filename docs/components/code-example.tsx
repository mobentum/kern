"use client";

import { useState } from "react";

const tabs = [
    {
        label: "REST API",
        title: "main.go",
        code: `package main

import "github.com/mobentum/kern"

type Task struct {
    ID        int    \`json:"id"\`
    Title     string \`json:"title" validate:"required,min=1"\`
    Completed bool   \`json:"completed"\`
}

func main() {
    app := kern.Default()          // logger + recovery
    app.Use(kern.CORS([]string{"*"}))

    api := app.Group("/api/v1")
    {
        api.GET("/tasks", listTasks)
        api.POST("/tasks", createTask)
        api.GET("/tasks/{id}", getTask)
        api.PATCH("/tasks/{id}", updateTask)
    }

    // graceful shutdown with 10s drain
    app.Run(":8080", kern.WithGracefulShutdown(10))
}

func listTasks(c *kern.Context) {
    c.OK(tasks)
}

func createTask(c *kern.Context) {
    var t Task
    if err := c.Bind(&t); err != nil {
        c.JSONError(422, err.Error())
        return
    }
    t.ID = nextID()
    tasks = append(tasks, t)
    c.Created(t)
}

func getTask(c *kern.Context) {
    id := c.Param("id")
    // lookup & return...
    c.OK(task)
}

func updateTask(c *kern.Context) {
    id := c.Param("id")
    var t Task
    if err := c.Bind(&t); err != nil {
        c.JSONError(422, err.Error())
        return
    }
    // update & return...
    c.OK(updated)
}`,
    },
    {
        label: "Custom Middleware",
        title: "middleware.go",
        code: `import (
    "log"
    "net/http"
    "time"

    "github.com/mobentum/kern"
)

// Add a request ID to every response
func RequestID() kern.MiddlewareFunc {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            id := r.Header.Get("X-Request-ID")
            if id == "" {
                id = generateID()
            }
            w.Header().Set("X-Request-ID", id)
            next.ServeHTTP(w, r)
        })
    }
}

// Timeout any handler after a deadline
func Timeout(d time.Duration) kern.MiddlewareFunc {
    return func(next http.Handler) http.Handler {
        return http.TimeoutHandler(next, d,
            "request timed out",
        )
    }
}

// Apply per-route for specific endpoints
app.RouteWithMiddleware("GET", "/slow-report",
    reportHandler,
    Timeout(5 * time.Second),
    RequestID(),
)`,
    },
    {
        label: "File Streaming",
        title: "files.go",
        code: `import "github.com/mobentum/kern"

func main() {
    app := kern.Default()

    // Stream any file with byte-range support
    app.GET("/download/{filename}", func(c *kern.Context) {
        name := c.Param("filename")
        c.DownloadFile("./uploads/" + name, name)
    })

    // Accept multipart file uploads
    app.POST("/upload", func(c *kern.Context) {
        file, err := c.File("document")
        if err != nil {
            c.JSONError(400, "missing file")
            return
        }

        // Save to disk
        if err := c.SaveFile(file, "./uploads/" + file.Filename); err != nil {
            c.JSONError(500, "save failed")
            return
        }

        c.Created(map[string]string{
            "filename": file.Filename,
            "size":     humanSize(file.Size),
        })
    })

    app.Run(":8080")
}`,
    },
];

export function CodeExampleSection() {
    const [activeTab, setActiveTab] = useState(0);

    return (
        <div className="relative py-16 bg-slate-950">
            <div className="max-w-7xl mx-auto px-6 lg:px-8">
                <div className="text-center mb-16">
                    <h2 className="text-3xl lg:text-5xl font-bold mb-4 bg-gradient-to-r from-violet-400 via-indigo-400 to-cyan-400 bg-clip-text text-transparent">
                        See Kern in action
                    </h2>
                    <p className="text-lg text-slate-400">
                        Real code, not hello world. Copy, paste, run.
                    </p>
                </div>

                <div className="max-w-3xl mx-auto">
                    {/* Tabs */}
                    <div className="flex gap-1 mb-0 bg-slate-900/80 rounded-t-xl border border-b-0 border-slate-800 p-1.5">
                        {tabs.map((tab, i) => (
                            <button
                                key={tab.label}
                                onClick={() => setActiveTab(i)}
                                className={`px-4 py-2 rounded-lg text-sm font-medium transition-all ${
                                    activeTab === i
                                        ? "bg-violet-600/20 text-violet-300"
                                        : "text-slate-500 hover:text-slate-300"
                                }`}
                            >
                                {tab.label}
                            </button>
                        ))}
                    </div>

                    {/* Code Block */}
                    <div className="relative bg-slate-900/80 backdrop-blur-sm rounded-b-xl border border-slate-800 shadow-2xl overflow-hidden">
                        {/* Terminal Header */}
                        <div className="flex items-center justify-between px-4 py-3 bg-slate-950/50 border-b border-slate-800">
                            <div className="flex items-center gap-2">
                                <div className="flex gap-1.5">
                                    <div className="w-3 h-3 rounded-full bg-red-500/80" />
                                    <div className="w-3 h-3 rounded-full bg-yellow-500/80" />
                                    <div className="w-3 h-3 rounded-full bg-green-500/80" />
                                </div>
                                <span className="text-xs text-slate-500 ml-2">{tabs[activeTab].title}</span>
                            </div>
                            <button
                                onClick={() => navigator.clipboard.writeText(tabs[activeTab].code)}
                                className="text-xs text-slate-500 hover:text-slate-300 transition-colors flex items-center gap-1"
                            >
                                <svg className="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M8 16H6a2 2 0 01-2-2V6a2 2 0 012-2h8a2 2 0 012 2v2m-6 12h8a2 2 0 002-2v-8a2 2 0 00-2-2h-8a2 2 0 00-2 2v8a2 2 0 002 2z" />
                                </svg>
                                Copy
                            </button>
                        </div>

                        {/* Code */}
                        <pre className="p-6 text-sm font-mono overflow-x-auto">
                            <code className="text-slate-300">{tabs[activeTab].code}</code>
                        </pre>
                    </div>

                    <p className="text-center text-sm text-slate-600 mt-6">
                        These examples work out of the box.{" "}
                        <a href="https://github.com/mobentum/kern/tree/main/examples" className="text-violet-400 hover:text-violet-300 transition-colors">
                            More examples on GitHub →
                        </a>
                    </p>
                </div>
            </div>
        </div>
    );
}
