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

// --- 資料結構 (保持不變) ---

type Task struct {
	ID          int       `json:"id"`
	Description string    `json:"description"`
	Completed   bool      `json:"completed"`
	CreatedAt   time.Time `json:"created_at"`
}

type ByDate []Task

func (a ByDate) Len() int           { return len(a) }
func (a ByDate) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByDate) Less(i, j int) bool { return a[i].CreatedAt.Before(a[j].CreatedAt) }

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

// --- 全域變數 ---
var todoList *TodoList

// --- HTML 模板 (前端介面) ---
// 為了方便，直接將 HTML/CSS 寫在這裡
const htmlTemplate = `
<!DOCTYPE html>
<html lang="zh-TW">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Go To-Do List</title>
    <style>
        body { font-family: 'Microsoft JhengHei', sans-serif; background-color: #f4f4f9; display: flex; justify-content: center; padding-top: 50px; }
        .container { background: white; padding: 2rem; border-radius: 8px; box-shadow: 0 4px 6px rgba(0,0,0,0.1); width: 500px; }
        h1 { text-align: center; color: #333; }
        .input-group { display: flex; gap: 10px; margin-bottom: 20px; }
        input[type="text"] { flex: 1; padding: 10px; border: 1px solid #ddd; border-radius: 4px; }
        button.add-btn { padding: 10px 20px; background-color: #28a745; color: white; border: none; border-radius: 4px; cursor: pointer; }
        button.add-btn:hover { background-color: #218838; }
        ul { list-style: none; padding: 0; }
        li { background: #fff; border-bottom: 1px solid #eee; padding: 10px; display: flex; align-items: center; justify-content: space-between; }
        li:last-child { border-bottom: none; }
        .task-content { display: flex; align-items: center; gap: 10px; flex: 1; }
        .completed { text-decoration: line-through; color: #888; }
        .time { font-size: 0.8em; color: #ccc; margin-left: 10px; }
        .actions a { text-decoration: none; color: #dc3545; margin-left: 10px; font-size: 0.9em; }
        .actions a:hover { text-decoration: underline; }
    </style>
</head>
<body>
    <div class="container">
        <h1>我的待辦清單</h1>
        
        <form action="/add" method="POST" class="input-group">
            <input type="text" name="description" placeholder="輸入新的待辦事項..." required>
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
                        {{.Description}} <span class="time">({{.CreatedAt.Format "01-02 15:04"}})</span>
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
</body>
</html>
`

// --- Handlers ---

func indexHandler(w http.ResponseWriter, r *http.Request) {
	sort.Sort(ByDate(todoList.Tasks))
	t, _ := template.New("todo").Parse(htmlTemplate)
	t.Execute(w, todoList)
}

func addHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		desc := r.FormValue("description")
		if desc != "" {
			task := Task{
				ID:          todoList.NextID,
				Description: desc,
				Completed:   false,
				CreatedAt:   time.Now(),
			}
			todoList.Tasks = append(todoList.Tasks, task)
			todoList.NextID++
			todoList.Save()
		}
	}
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func toggleHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		id, _ := strconv.Atoi(r.FormValue("id"))
		for i := range todoList.Tasks {
			if todoList.Tasks[i].ID == id {
				todoList.Tasks[i].Completed = !todoList.Tasks[i].Completed
				todoList.Save()
				break
			}
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

	fmt.Println("=== 網站伺服器已啟動 ===")
	fmt.Println("請打開瀏覽器訪問: http://localhost:8080")
	fmt.Println("按 Ctrl+C 結束程式")

	// 這裡可能會報錯如果不允許防火牆，請允許
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		log.Fatal("無法啟動伺服器: ", err)
	}
}
