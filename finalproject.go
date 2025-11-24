package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"sort"
	"strconv"
	"time"
)

type Task struct {
	ID          int       `json:"id"`
	Description string    `json:"description"`
	Completed   bool      `json:"completed"`
	CreatedAt   time.Time `json:"created_at"`
	DueAt       time.Time `json:"due_at"`
}

type ByDueDate []Task

func (a ByDueDate) Len() int           { return len(a) }
func (a ByDueDate) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByDueDate) Less(i, j int) bool { return a[i].DueAt.Before(a[j].DueAt) }

type TodoList struct {
	Tasks    []Task `json:"tasks"`
	NextID   int    `json:"next_id"`
	filename string
}

func NewTodoList(filename string) *TodoList {
	tl := &TodoList{Tasks: []Task{}, NextID: 1, filename: filename}
	tl.Load()
	return tl
}

func (tl *TodoList) Load() {
	file, err := os.ReadFile(tl.filename)
	if err == nil && len(file) > 0 {
		json.Unmarshal(file, tl)
	}
}

func (tl *TodoList) Save() {
	data, _ := json.MarshalIndent(tl, "", "  ")
	os.WriteFile(tl.filename, data, 0644)
}

var todoList *TodoList

func remainingTime(d time.Time) string {
	now := time.Now()
	diff := d.Sub(now)

	if diff > 0 {
		if diff.Hours() >= 24 {
			return fmt.Sprintf("剩 %.0f 天", diff.Hours()/24)
		}
		if diff.Hours() >= 1 {
			return fmt.Sprintf("剩 %.0f 小時", diff.Hours())
		}
		return fmt.Sprintf("剩 %.0f 分鐘", diff.Minutes())
	}

	diff = now.Sub(d)
	if diff.Hours() >= 24 {
		return fmt.Sprintf("已逾期 %.0f 天", diff.Hours()/24)
	}
	if diff.Hours() >= 1 {
		return fmt.Sprintf("已逾期 %.0f 小時", diff.Hours())
	}
	return fmt.Sprintf("已逾期 %.0f 分鐘", diff.Minutes())
}

const htmlTemplate = `
<!DOCTYPE html>
<html lang="zh-TW">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>Go To-Do List</title>
<style>
body { font-family: 'Microsoft JhengHei', sans-serif; background-color: #f4f4f9; display: flex; justify-content: center; padding-top: 50px; }
.container { background: white; padding: 2rem; border-radius: 8px; box-shadow: 0 4px 6px rgba(0,0,0,0.1); width: 520px; }
h1 { text-align: center; color: #333; }
.input-group { display: flex; gap: 10px; margin-bottom: 20px; }
input[type="text"], input[type="datetime-local"] { padding: 10px; border: 1px solid #ddd; border-radius: 4px; }
input[type="text"] { flex: 1; }
button.add-btn { padding: 10px 20px; background-color: #28a745; color: white; border: none; border-radius: 4px; cursor: pointer; }
button.add-btn:hover { background-color: #218838; }
ul { list-style: none; padding: 0; }
li { background: #fff; border-bottom: 1px solid #eee; padding: 10px; display: flex; align-items: center; justify-content: space-between; }
.task-content { display: flex; align-items: center; gap: 10px; flex: 1; }
.completed { text-decoration: line-through; color: #888; }
.time { font-size: 0.8em; margin-left: 10px; }
.red { color: red; }
.actions a { text-decoration: none; color: #dc3545; margin-left: 10px; font-size: 0.9em; }
.actions a:hover { text-decoration: underline; }
</style>
</head>
<body>
<div class="container">
<h1>我的待辦清單</h1>

<form action="/add" method="POST" class="input-group">
    <input type="text" name="description" placeholder="輸入新的待辦事項..." required>
    <input type="datetime-local" name="due_at" required>
    <button type="submit" class="add-btn">新增</button>
</form>

<ul>
{{range .Tasks}}
<li>
    <div class="task-content">
        <form action="/toggle" method="POST" style="margin:0;">
            <input type="hidden" name="id" value="{{.ID}}">
            <input type="checkbox" onchange="this.form.submit()" {{if .Completed}}checked{{end}}>
        </form>

        <span class="{{if .Completed}}completed{{end}}">
            {{.Description}}
            <span class="time {{if .DueAt.Before now}}red{{end}}">
                到期：{{.DueAt.Format "01-02 15:04"}} ｜ {{remain .DueAt}}
            </span>
        </span>
    </div>

    <div class="actions">
        <a href="/delete?id={{.ID}}">刪除</a>
    </div>
</li>
{{else}}
<li style="justify-content: center; color: #888;">目前沒有任務 🎉</li>
{{end}}
</ul>

</div>

<script>
setTimeout(function(){ location.reload(); }, 60000);
</script>
</body>
</html>
`

func indexHandler(w http.ResponseWriter, r *http.Request) {
	sort.Sort(ByDueDate(todoList.Tasks))

	funcMap := template.FuncMap{
		"remain": remainingTime,
		"now":    time.Now,
	}

	t, _ := template.New("todo").Funcs(funcMap).Parse(htmlTemplate)
	t.Execute(w, todoList)
}

func addHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		desc := r.FormValue("description")
		dueStr := r.FormValue("due_at")
		dueAt, _ := time.Parse("2006-01-02T15:04", dueStr)

		task := Task{
			ID:          todoList.NextID,
			Description: desc,
			Completed:   false,
			CreatedAt:   time.Now(),
			DueAt:       dueAt,
		}

		todoList.Tasks = append(todoList.Tasks, task)
		todoList.NextID++
		todoList.Save()
	}
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func toggleHandler(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.Atoi(r.FormValue("id"))
	for i := range todoList.Tasks {
		if todoList.Tasks[i].ID == id {
			todoList.Tasks[i].Completed = !todoList.Tasks[i].Completed
			todoList.Save()
			break
		}
	}
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func deleteHandler(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.Atoi(r.URL.Query().Get("id"))
	for i, task := range todoList.Tasks {
		if task.ID == id {
			todoList.Tasks = append(todoList.Tasks[:i], todoList.Tasks[i+1:]...)
			todoList.Save()
			break
		}
	}
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func main() {
	todoList = NewTodoList("todos.json")

	http.HandleFunc("/", indexHandler)
	http.HandleFunc("/add", addHandler)
	http.HandleFunc("/toggle", toggleHandler)
	http.HandleFunc("/delete", deleteHandler)

	fmt.Println("Server started at http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
