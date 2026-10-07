package service

import (
	"github.com/bikojii/todo-app"
	"github.com/bikojii/todo-app/pkg/repository"
)

type TodoItemService struct {
	repo     repository.TodoItem
	listRepo repository.TodoList
}

func NewTodoItemService(repo repository.TodoItem, listRepo repository.TodoList) *TodoItemService {
	return &TodoItemService{repo: repo, listRepo: listRepo}
}

func (s *TodoItemService) Create(userId int, listId int, item todo.TodoItem) (int, error) {
	if err := item.Validate(); err != nil {
		return 0, err
	}
	return s.repo.Create(userId, listId, item)
}

func (s *TodoItemService) GetAll(userId int, listId int) ([]todo.TodoItem, error) {
	if _, err := s.listRepo.GetById(userId, listId); err != nil {
		return nil, err
	}
	return s.repo.GetAll(userId, listId)
}

func (s *TodoItemService) GetById(userId int, itemId int) (todo.TodoItem, error) {
	return s.repo.GetById(userId, itemId)
}

func (s *TodoItemService) Delete(userId int, itemId int) error {
	return s.repo.Delete(userId, itemId)
}

func (s *TodoItemService) Update(userId, itemId int, input todo.UpdateItemInput) error {
	if err := input.Validate(); err != nil {
		return err
	}
	return s.repo.Update(userId, itemId, input)
}
