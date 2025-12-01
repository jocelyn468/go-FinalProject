package main

import (
	"crypto/sha256"
	"encoding/hex"
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

// --- 資料結構定義 ---

type User struct {
	Username     string `json:"username"`
	PasswordHash string `json:"password_hash"`
}

type Task struct {
	ID          int       `json:"id"`
	Description string    `json:"description"`
	Completed   bool      `json:"completed"`
	CreatedAt   time.Time `json:"created_at"`
	DueAt       time.Time `json:"due_at"`
	Username    string    `json:"username"`
}

type AppData struct {
	Users  []User `json:"users"`
	Tasks  []Task `json:"tasks"`
	NextID int    `json:"next_id"`
}

// --- 全域變數 ---

var appData *AppData
var sessions = make(map[string]string) // sessionID -> username

// --- 輔助函式 ---

func hashPassword(password string) string {
	hash := sha256.Sum256([]byte(password))
	return hex.EncodeToString(hash[:])
}

func loadData() {
	file, err := os.ReadFile("app_data.json")
	if err == nil && len(file) > 0 {
		json.Unmarshal(file, appData)
	}
}

func saveData() {
	data, _ := json.MarshalIndent(appData, "", "  ")
	os.WriteFile("app_data.json", data, 0644)
}

func getUsername(r *http.Request) string {
	cookie, err := r.Cookie("session")
	if err != nil {
		return ""
	}
	return sessions[cookie.Value]
}

func requireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if getUsername(r) == "" {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}
		next(w, r)
	}
}

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

// --- HTML 模板 ---

const loginTemplate = `
<!DOCTYPE html>
<html lang="zh-TW">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>登入 - To-Do List</title>
<style>
body { font-family: 'Microsoft JhengHei', sans-serif; background: linear-gradient(135deg, #667eea 0%, #764ba2 100%); display: flex; justify-content: center; align-items: center; height: 100vh; margin: 0; }
.container { background: white; padding: 2rem; border-radius: 12px; box-shadow: 0 8px 16px rgba(0,0,0,0.2); width: 360px; }
h1 { text-align: center; color: #333; margin-bottom: 1.5rem; }
.form-group { margin-bottom: 1rem; }
label { display: block; margin-bottom: 0.5rem; color: #555; font-weight: 500; }
input[type="text"], input[type="password"] { width: 100%; padding: 10px; border: 1px solid #ddd; border-radius: 4px; box-sizing: border-box; font-size: 14px; }
button { width: 100%; padding: 12px; background-color: #667eea; color: white; border: none; border-radius: 4px; cursor: pointer; font-size: 16px; font-weight: 500; margin-top: 1rem; }
button:hover { background-color: #5568d3; }
.switch { text-align: center; margin-top: 1rem; color: #666; }
.switch a { color: #667eea; text-decoration: none; font-weight: 500; }
.switch a:hover { text-decoration: underline; }
.error { color: #dc3545; text-align: center; margin-bottom: 1rem; font-size: 14px; }
</style>
</head>
<body>
<div class="container">
<h1>{{if .IsRegister}}註冊帳號{{else}}登入系統{{end}}</h1>
{{if .Error}}<div class="error">{{.Error}}</div>{{end}}

<form method="POST">
    <div class="form-group">
        <label>使用者名稱</label>
        <input type="text" name="username" required autofocus>
    </div>
    <div class="form-group">
        <label>密碼</label>
        <input type="password" name="password" required>
    </div>
    <button type="submit">{{if .IsRegister}}註冊{{else}}登入{{end}}</button>
</form>

<div class="switch">
    {{if .IsRegister}}
        已有帳號？<a href="/login">前往登入</a>
    {{else}}
        還沒帳號？<a href="/register">立即註冊</a>
    {{end}}
</div>
</div>
</body>
</html>
`

const listTemplate = `
<!DOCTYPE html>
<html lang="zh-TW">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>我的待辦清單</title>
<style>
body { font-family: 'Microsoft JhengHei', sans-serif; background-color: #f4f4f9; margin: 0; padding-top: 20px; }
.header { background: linear-gradient(135deg, #667eea 0%, #764ba2 100%); color: white; padding: 1.5rem; margin-bottom: 20px; box-shadow: 0 2px 4px rgba(0,0,0,0.1); }
.header-content { max-width: 800px; margin: 0 auto; display: flex; justify-content: space-between; align-items: center; }
.header h1 { margin: 0; font-size: 1.8rem; }
.user-info { display: flex; gap: 15px; align-items: center; }
.username { font-size: 1rem; }
.nav-links a { color: white; text-decoration: none; padding: 8px 15px; border-radius: 4px; background: rgba(255,255,255,0.2); transition: background 0.3s; }
.nav-links a:hover { background: rgba(255,255,255,0.3); }
.container { max-width: 800px; margin: 0 auto; padding: 0 1rem; }
.view-toggle { display: flex; gap: 10px; margin-bottom: 20px; justify-content: center; }
.view-toggle a { padding: 10px 20px; background: white; color: #667eea; text-decoration: none; border-radius: 4px; box-shadow: 0 2px 4px rgba(0,0,0,0.1); transition: all 0.3s; }
.view-toggle a:hover, .view-toggle a.active { background: #667eea; color: white; }
.input-group { display: flex; gap: 10px; margin-bottom: 20px; background: white; padding: 1.5rem; border-radius: 8px; box-shadow: 0 2px 6px rgba(0,0,0,0.1); }
input[type="text"], input[type="datetime-local"] { padding: 10px; border: 1px solid #ddd; border-radius: 4px; }
input[type="text"] { flex: 1; }
button.add-btn { padding: 10px 20px; background-color: #28a745; color: white; border: none; border-radius: 4px; cursor: pointer; font-weight: 500; }
button.add-btn:hover { background-color: #218838; }
.task-list { background: white; border-radius: 8px; box-shadow: 0 2px 6px rgba(0,0,0,0.1); }
ul { list-style: none; padding: 0; margin: 0; }
li { border-bottom: 1px solid #eee; padding: 15px; display: flex; align-items: center; justify-content: space-between; }
li:last-child { border-bottom: none; }
.task-content { display: flex; align-items: center; gap: 10px; flex: 1; }
.completed { text-decoration: line-through; color: #888; }
.time { font-size: 0.85em; margin-left: 10px; color: #666; }
.red { color: #dc3545; font-weight: 500; }
.actions a { text-decoration: none; color: #dc3545; margin-left: 10px; font-size: 0.9em; }
.actions a:hover { text-decoration: underline; }
.empty-state { text-align: center; padding: 3rem; color: #888; font-size: 1.1rem; }
.filter-tabs { display: flex; gap: 10px; margin-bottom: 15px; justify-content: center; }
.filter-tabs a { padding: 5px 15px; border-radius: 15px; text-decoration: none; font-size: 0.9rem; color: #555; background: #e9ecef; }
.filter-tabs a.active { background: #667eea; color: white; }
</style>
</head>
<body>
<div class="header">
    <div class="header-content">
        <h1>📝 我的待辦清單</h1>
        <div class="user-info">
            <span class="username">👤 {{.Username}}</span>
            <div class="nav-links">
                <a href="/logout">登出</a>
            </div>
        </div>
    </div>
</div>

<div class="container">
    <div style="text-align:center; margin-bottom:15px;">
        {{if gt .OverdueCount 0}}
            <span style="color:#dc3545; font-weight:500;">⚠️ 你有 {{.OverdueCount}} 個逾期任務</span>
        {{end}}
    </div>

    <div class="view-toggle">
        <a href="/" class="active">📋 清單模式</a>
        <a href="/calendar">📅 月曆模式</a>
    </div>

    <div class="filter-tabs">
        <a href="/?filter=" class="{{if eq .Filter ""}}active{{end}}">全部</a>
        <a href="/?filter=today" class="{{if eq .Filter "today"}}active{{end}}">今日任務</a>
        <a href="/?filter=incomplete" class="{{if eq .Filter "incomplete"}}active{{end}}">未完成</a>
    </div>

    <form action="/add" method="POST" class="input-group">
        <input type="text" name="description" placeholder="輸入新的待辦事項..." required>
        <input type="datetime-local" name="due_at" required max="9999-12-31T23:59">
        <button type="submit" class="add-btn">新增</button>
    </form>

    <div class="task-list">
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
        <li class="empty-state">目前沒有任務 🎉</li>
        {{end}}
        </ul>
    </div>
</div>

<script>
setTimeout(function(){ location.reload(); }, 60000);
</script>
</body>
</html>
`
const calendarTemplate = `
<!DOCTYPE html>
<html lang="zh-TW">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>月曆 - 待辦清單</title>
<style>
body { font-family: 'Microsoft JhengHei', sans-serif; background-color: #f4f4f9; margin: 0; padding-top: 20px; }
.header { background: linear-gradient(135deg, #667eea 0%, #764ba2 100%); color: white; padding: 1.5rem; margin-bottom: 20px; box-shadow: 0 2px 4px rgba(0,0,0,0.1); }
.header-content { max-width: 1200px; margin: 0 auto; display: flex; justify-content: space-between; align-items: center; }
.header h1 { margin: 0; font-size: 1.8rem; }
.user-info { display: flex; gap: 15px; align-items: center; }
.username { font-size: 1rem; }
.nav-links a { color: white; text-decoration: none; padding: 8px 15px; border-radius: 4px; background: rgba(255,255,255,0.2); transition: background 0.3s; }
.nav-links a:hover { background: rgba(255,255,255,0.3); }
.container { max-width: 1200px; margin: 0 auto; padding: 0 1rem; }
.view-toggle { display: flex; gap: 10px; margin-bottom: 20px; justify-content: center; }
.view-toggle a { padding: 10px 20px; background: white; color: #667eea; text-decoration: none; border-radius: 4px; box-shadow: 0 2px 4px rgba(0,0,0,0.1); transition: all 0.3s; }
.view-toggle a:hover, .view-toggle a.active { background: #667eea; color: white; }
.calendar-nav { display: flex; justify-content: space-between; align-items: center; background: white; padding: 1rem; border-radius: 8px; margin-bottom: 20px; box-shadow: 0 2px 6px rgba(0,0,0,0.1); }
.calendar-nav a { text-decoration: none; color: #667eea; padding: 8px 15px; border-radius: 4px; background: #f0f0f0; }
.calendar-nav a:hover { background: #e0e0e0; }
.calendar-nav h2 { margin: 0; color: #333; }
.calendar { background: white; border-radius: 8px; box-shadow: 0 2px 6px rgba(0,0,0,0.1); padding: 1rem; }
.calendar-grid { display: grid; grid-template-columns: repeat(7, 1fr); gap: 1px; background: #ddd; border: 1px solid #ddd; }
.calendar-header { background: #667eea; color: white; padding: 10px; text-align: center; font-weight: 600; }
.calendar-day { background: white; padding: 8px; min-height: 100px; position: relative; }
.calendar-day.other-month { background: #f9f9f9; }
.calendar-day.other-month .day-number { color: #bbb; }
.calendar-day.today { background: #fff3cd; }
.day-number { font-weight: 600; margin-bottom: 5px; color: #333; }
.day-task { font-size: 0.75em; padding: 2px 4px; margin: 2px 0; background: #e7f3ff; border-radius: 3px; cursor: pointer; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.day-task.completed { background: #d4edda; text-decoration: line-through; color: #666; }
.day-task.overdue { background: #f8d7da; color: #721c24; }
.task-detail { position: fixed; top: 50%; left: 50%; transform: translate(-50%, -50%); background: white; padding: 1.5rem; border-radius: 8px; box-shadow: 0 4px 12px rgba(0,0,0,0.3); z-index: 1000; min-width: 300px; display: none; }
.overlay { position: fixed; top: 0; left: 0; right: 0; bottom: 0; background: rgba(0,0,0,0.5); z-index: 999; display: none; }
.task-detail h3 { margin-top: 0; color: #333; }
.task-detail-actions { display: flex; gap: 10px; margin-top: 1rem; }
.task-detail-actions a, .task-detail-actions button { padding: 8px 15px; border-radius: 4px; text-decoration: none; cursor: pointer; border: none; font-size: 14px; }
.close-btn { background: #6c757d; color: white; }
.delete-btn { background: #dc3545; color: white; }
</style>
</head>
<body>
<div class="header">
    <div class="header-content">
        <h1>📅 月曆模式</h1>
        <div class="user-info">
            <span class="username">👤 {{.Username}}</span>
            <div class="nav-links">
                <a href="/logout">登出</a>
            </div>
        </div>
    </div>
</div>

<div class="container">
    <div class="view-toggle">
        <a href="/">📋 清單模式</a>
        <a href="/calendar" class="active">📅 月曆模式</a>
    </div>

    <div class="calendar-nav">
        <a href="/calendar?year={{.PrevYear}}&month={{.PrevMonth}}">← 上個月</a>
        <h2>{{printf "%d" .Year}} 年 {{printf "%d" .Month}} 月</h2>
        <a href="/calendar?year={{.NextYear}}&month={{.NextMonth}}">下個月 →</a>
    </div>

    <div class="calendar">
        <div class="calendar-grid">
            <div class="calendar-header">日</div>
            <div class="calendar-header">一</div>
            <div class="calendar-header">二</div>
            <div class="calendar-header">三</div>
            <div class="calendar-header">四</div>
            <div class="calendar-header">五</div>
            <div class="calendar-header">六</div>
            
            {{range .Days}}
            <div class="calendar-day {{.Class}}">
                <div class="day-number">{{.Day}}</div>
                {{range .Tasks}}
                <div class="day-task {{if .Completed}}completed{{else if .IsOverdue}}overdue{{end}}" 
                     onclick="showTask({{.ID}}, '{{.Description}}', '{{.DueAt.Format "2006-01-02 15:04"}}', {{.Completed}})">
                    {{.Description}}
                </div>
                {{end}}
            </div>
            {{end}}
        </div>
    </div>
</div>

<div class="overlay" id="overlay" onclick="closeTask()"></div>
<div class="task-detail" id="taskDetail">
    <h3 id="taskTitle"></h3>
    <p><strong>到期時間：</strong><span id="taskDue"></span></p>
    <p><strong>狀態：</strong><span id="taskStatus"></span></p>
    <div class="task-detail-actions">
        <button class="close-btn" onclick="closeTask()">關閉</button>
        <a id="deleteLink" class="delete-btn">刪除</a>
    </div>
</div>

<script>
function showTask(id, description, dueAt, completed) {
    document.getElementById('taskTitle').textContent = description;
    document.getElementById('taskDue').textContent = dueAt;
    document.getElementById('taskStatus').textContent = completed ? '✅ 已完成' : '⏳ 待完成';
    document.getElementById('deleteLink').href = '/delete?id=' + id;
    document.getElementById('overlay').style.display = 'block';
    document.getElementById('taskDetail').style.display = 'block';
}

function closeTask() {
    document.getElementById('overlay').style.display = 'none';
    document.getElementById('taskDetail').style.display = 'none';
}
</script>
</body>
</html>
`

// --- Handlers ---

func loginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		username := r.FormValue("username")
		password := r.FormValue("password")
		passwordHash := hashPassword(password)

		for _, user := range appData.Users {
			if user.Username == username && user.PasswordHash == passwordHash {
				sessionID := fmt.Sprintf("%d", time.Now().UnixNano())
				sessions[sessionID] = username
				http.SetCookie(w, &http.Cookie{
					Name:  "session",
					Value: sessionID,
					Path:  "/",
				})
				http.Redirect(w, r, "/", http.StatusSeeOther)
				return
			}
		}

		data := map[string]interface{}{
			"IsRegister": false,
			"Error":      "使用者名稱或密碼錯誤",
		}
		t, _ := template.New("login").Parse(loginTemplate)
		t.Execute(w, data)
		return
	}

	data := map[string]interface{}{"IsRegister": false}
	t, _ := template.New("login").Parse(loginTemplate)
	t.Execute(w, data)
}

func registerHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		username := r.FormValue("username")
		password := r.FormValue("password")

		for _, user := range appData.Users {
			if user.Username == username {
				data := map[string]interface{}{
					"IsRegister": true,
					"Error":      "使用者名稱已存在",
				}
				t, _ := template.New("login").Parse(loginTemplate)
				t.Execute(w, data)
				return
			}
		}

		newUser := User{
			Username:     username,
			PasswordHash: hashPassword(password),
		}
		appData.Users = append(appData.Users, newUser)
		saveData()

		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	data := map[string]interface{}{"IsRegister": true}
	t, _ := template.New("login").Parse(loginTemplate)
	t.Execute(w, data)
}

func logoutHandler(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("session")
	if err == nil {
		delete(sessions, cookie.Value)
	}
	http.SetCookie(w, &http.Cookie{
		Name:   "session",
		Value:  "",
		Path:   "/",
		MaxAge: -1,
	})
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	username := getUsername(r)
	filter := r.URL.Query().Get("filter") // 取得過濾參數

	var userTasks []Task
	now := time.Now()

	// 篩選任務
	for _, task := range appData.Tasks {
		if task.Username == username {
			if filter == "today" {
				if task.DueAt.Format("2006-01-02") != now.Format("2006-01-02") {
					continue
				}
			} else if filter == "incomplete" {
				if task.Completed {
					continue
				}
			}
			userTasks = append(userTasks, task)
		}
	}

	// 智慧排序：逾期且未完成的優先 -> 接著按到期時間
	sort.SliceStable(userTasks, func(i, j int) bool {
		iOver := userTasks[i].DueAt.Before(now) && !userTasks[i].Completed
		jOver := userTasks[j].DueAt.Before(now) && !userTasks[j].Completed

		if iOver != jOver {
			return iOver // 如果一個逾期一個沒逾期，逾期的排前面
		}
		return userTasks[i].DueAt.Before(userTasks[j].DueAt) // 否則按時間排
	})

	// 計算總逾期數（不管過濾條件，算給 Header 警告用的）
	overdueCount := 0
	for _, task := range appData.Tasks {
		if task.Username == username && task.DueAt.Before(now) && !task.Completed {
			overdueCount++
		}
	}

	funcMap := template.FuncMap{
		"remain": remainingTime,
		"now":    time.Now,
	}

	data := map[string]interface{}{
		"Username":     username,
		"Tasks":        userTasks,
		"IsCalendar":   false,
		"OverdueCount": overdueCount,
		"Filter":       filter,
	}

	t, _ := template.New("list").Funcs(funcMap).Parse(listTemplate)
	t.Execute(w, data)
}

func calendarHandler(w http.ResponseWriter, r *http.Request) {
	username := getUsername(r)

	year, _ := strconv.Atoi(r.URL.Query().Get("year"))
	month, _ := strconv.Atoi(r.URL.Query().Get("month"))

	if year == 0 {
		now := time.Now()
		year = now.Year()
		month = int(now.Month())
	}

	firstDay := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.Local)
	startWeekday := int(firstDay.Weekday())
	startDate := firstDay.AddDate(0, 0, -startWeekday)

	var days []map[string]interface{}
	currentDate := startDate
	now := time.Now()

	for i := 0; i < 42; i++ {
		var dayTasks []map[string]interface{}
		for _, task := range appData.Tasks {
			if task.Username == username {
				taskDate := task.DueAt.Format("2006-01-02")
				currentDateStr := currentDate.Format("2006-01-02")
				if taskDate == currentDateStr {
					dayTasks = append(dayTasks, map[string]interface{}{
						"ID":          task.ID,
						"Description": task.Description,
						"Completed":   task.Completed,
						"DueAt":       task.DueAt,
						"IsOverdue":   task.DueAt.Before(now) && !task.Completed,
					})
				}
			}
		}

		class := ""
		if currentDate.Year() != year || int(currentDate.Month()) != month {
			class = "other-month"
		}
		if currentDate.Format("2006-01-02") == now.Format("2006-01-02") {
			class = "today"
		}

		days = append(days, map[string]interface{}{
			"Day":   currentDate.Day(),
			"Tasks": dayTasks,
			"Class": class,
		})

		currentDate = currentDate.AddDate(0, 0, 1)
	}

	prevMonth := month - 1
	prevYear := year
	if prevMonth == 0 {
		prevMonth = 12
		prevYear--
	}

	nextMonth := month + 1
	nextYear := year
	if nextMonth == 13 {
		nextMonth = 1
		nextYear++
	}

	data := map[string]interface{}{
		"Username":  username,
		"Year":      year,
		"Month":     month,
		"Days":      days,
		"PrevYear":  prevYear,
		"PrevMonth": prevMonth,
		"NextYear":  nextYear,
		"NextMonth": nextMonth,
	}

	t, _ := template.New("calendar").Parse(calendarTemplate)
	t.Execute(w, data)
}

func addHandler(w http.ResponseWriter, r *http.Request) {
	username := getUsername(r)
	if r.Method == "POST" {
		desc := r.FormValue("description")
		dueStr := r.FormValue("due_at")
		dueAt, _ := time.Parse("2006-01-02T15:04", dueStr)

		task := Task{
			ID:          appData.NextID,
			Description: desc,
			Completed:   false,
			CreatedAt:   time.Now(),
			DueAt:       dueAt,
			Username:    username,
		}

		appData.Tasks = append(appData.Tasks, task)
		appData.NextID++
		saveData()
	}

	referer := r.Header.Get("Referer")
	if referer == "" {
		referer = "/"
	}
	http.Redirect(w, r, referer, http.StatusSeeOther)
}

func toggleHandler(w http.ResponseWriter, r *http.Request) {
	username := getUsername(r)
	id, _ := strconv.Atoi(r.FormValue("id"))
	for i := range appData.Tasks {
		if appData.Tasks[i].ID == id && appData.Tasks[i].Username == username {
			appData.Tasks[i].Completed = !appData.Tasks[i].Completed
			saveData()
			break
		}
	}
	http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
}

func deleteHandler(w http.ResponseWriter, r *http.Request) {
	username := getUsername(r)
	id, _ := strconv.Atoi(r.URL.Query().Get("id"))
	for i, task := range appData.Tasks {
		if task.ID == id && task.Username == username {
			appData.Tasks = append(appData.Tasks[:i], appData.Tasks[i+1:]...)
			saveData()
			break
		}
	}
	http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
}

// --- Main ---

func main() {
	appData = &AppData{
		Users:  []User{},
		Tasks:  []Task{},
		NextID: 1,
	}
	loadData()

	http.HandleFunc("/login", loginHandler)
	http.HandleFunc("/register", registerHandler)
	http.HandleFunc("/logout", logoutHandler)
	http.HandleFunc("/", requireAuth(indexHandler))
	http.HandleFunc("/calendar", requireAuth(calendarHandler))
	http.HandleFunc("/add", requireAuth(addHandler))
	http.HandleFunc("/toggle", requireAuth(toggleHandler))
	http.HandleFunc("/delete", requireAuth(deleteHandler))

	fmt.Println("Server started at http://localhost:8080")
	fmt.Println("請先註冊帳號再登入使用")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
