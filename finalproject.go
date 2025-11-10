package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Task struct {
	ID          int       `json:"id"`
	Description string    `json:"description"`
	Completed   bool      `json:"completed"`
	CreatedAt   time.Time `json:"created_at"`
}

type TodoList struct {
	Tasks    []Task `json:"tasks"`
	NextID   int    `json:"next_id"`
	filename string
}

func NewTodoList(filename string) *TodoList {
	tl := &TodoList{
		Tasks:    []Task{},
		NextID:   1,
		filename: filename,
	}
	tl.Load()
	return tl
}

func (tl *TodoList) Load() error {
	file, err := os.ReadFile(tl.filename)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	if len(file) == 0 {
		return nil
	}

	return json.Unmarshal(file, tl)
}

func (tl *TodoList) Save() error {
	data, err := json.MarshalIndent(tl, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(tl.filename, data, 0644)
}

func (tl *TodoList) Add(description string) {
	task := Task{
		ID:          tl.NextID,
		Description: description,
		Completed:   false,
		CreatedAt:   time.Now(),
	}
	tl.Tasks = append(tl.Tasks, task)
	tl.NextID++
	tl.Save()
	fmt.Printf("✓ 已添加任務 #%d: %s\n", task.ID, task.Description)
}

func (tl *TodoList) View() {
	if len(tl.Tasks) == 0 {
		fmt.Println("目前沒有任何待辦事項！")
		return
	}

	fmt.Println("\n=== 待辦事項列表 ===")
	for _, task := range tl.Tasks {
		status := "[ ]"
		if task.Completed {
			status = "[✓]"
		}
		fmt.Printf("%s #%d: %s (建立於: %s)\n",
			status,
			task.ID,
			task.Description,
			task.CreatedAt.Format("2006-01-02 15:04"))
	}
	fmt.Println()
}

func (tl *TodoList) Delete(id int) error {
	for i, task := range tl.Tasks {
		if task.ID == id {
			tl.Tasks = append(tl.Tasks[:i], tl.Tasks[i+1:]...)
			tl.Save()
			fmt.Printf("✓ 已刪除任務 #%d\n", id)
			return nil
		}
	}
	return fmt.Errorf("找不到 ID 為 %d 的任務", id)
}

func (tl *TodoList) Edit(id int, newDescription string) error {
	for i, task := range tl.Tasks {
		if task.ID == id {
			tl.Tasks[i].Description = newDescription
			tl.Save()
			fmt.Printf("✓ 已更新任務 #%d: %s\n", id, newDescription)
			return nil
		}
	}
	return fmt.Errorf("找不到 ID 為 %d 的任務", id)
}

func (tl *TodoList) Toggle(id int) error {
	for i, task := range tl.Tasks {
		if task.ID == id {
			tl.Tasks[i].Completed = !tl.Tasks[i].Completed
			tl.Save()
			status := "未完成"
			if tl.Tasks[i].Completed {
				status = "已完成"
			}
			fmt.Printf("✓ 任務 #%d 已標記為%s\n", id, status)
			return nil
		}
	}
	return fmt.Errorf("找不到 ID 為 %d 的任務", id)
}

func showHelp() {
	fmt.Println("\n=== To-Do List 指令說明 ===")
	fmt.Println("view           - 查看所有待辦事項")
	fmt.Println("add <內容>     - 添加新的待辦事項")
	fmt.Println("delete <ID>    - 刪除指定的待辦事項")
	fmt.Println("edit <ID> <內容> - 編輯指定的待辦事項")
	fmt.Println("toggle <ID>    - 切換任務完成狀態")
	fmt.Println("help           - 顯示此幫助信息")
	fmt.Println("exit           - 退出程式")
	fmt.Println()
}

func main() {
	todoList := NewTodoList("todos.json")
	reader := bufio.NewReader(os.Stdin)

	fmt.Println("=== 歡迎使用 Go To-Do List ===")
	fmt.Println("輸入 'help' 查看可用指令")

	for {
		fmt.Print("\n> ")
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)

		if input == "" {
			continue
		}

		parts := strings.SplitN(input, " ", 3)
		command := strings.ToLower(parts[0])

		switch command {
		case "view", "v":
			todoList.View()

		case "add", "a":
			if len(parts) < 2 {
				fmt.Println("❌ 請輸入任務內容。用法: add <內容>")
				continue
			}
			description := strings.Join(parts[1:], " ")
			todoList.Add(description)

		case "delete", "del", "d":
			if len(parts) < 2 {
				fmt.Println("❌ 請輸入任務 ID。用法: delete <ID>")
				continue
			}
			id, err := strconv.Atoi(parts[1])
			if err != nil {
				fmt.Println("❌ 無效的 ID")
				continue
			}
			if err := todoList.Delete(id); err != nil {
				fmt.Printf("❌ %v\n", err)
			}

		case "edit", "e":
			if len(parts) < 3 {
				fmt.Println("❌ 用法: edit <ID> <新內容>")
				continue
			}
			id, err := strconv.Atoi(parts[1])
			if err != nil {
				fmt.Println("❌ 無效的 ID")
				continue
			}
			newDescription := parts[2]
			if err := todoList.Edit(id, newDescription); err != nil {
				fmt.Printf("❌ %v\n", err)
			}

		case "toggle", "t":
			if len(parts) < 2 {
				fmt.Println("❌ 請輸入任務 ID。用法: toggle <ID>")
				continue
			}
			id, err := strconv.Atoi(parts[1])
			if err != nil {
				fmt.Println("❌ 無效的 ID")
				continue
			}
			if err := todoList.Toggle(id); err != nil {
				fmt.Printf("❌ %v\n", err)
			}

		case "help", "h":
			showHelp()

		case "exit", "quit", "q":
			fmt.Println("再見！")
			return

		default:
			fmt.Printf("❌ 未知的指令: %s\n", command)
			fmt.Println("輸入 'help' 查看可用指令")
		}
	}
}
