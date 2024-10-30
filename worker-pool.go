package main

import (
	"fmt"
	"sync"
	"time"
)

const JOBS = 20       // Количество задач
const SLEEPTIME = 500 // Время выполнения одной задачи (в мс)

type Pool struct {
	jobsChan chan string
	Delete   chan struct{}
	Add      chan struct{}
	workers  []*Worker
	wg       sync.WaitGroup // wg для
	sendWg   sync.WaitGroup // wg для избежания паники при записи в закрытый канал
}

type Worker struct {
	id   int
	done chan struct{}
	pool *Pool
}

func (w *Worker) Start() {
	for {
		select {
		case <-w.done:
			return
		case s, ok := <-w.pool.jobsChan:
			if !ok {
				return
			}
			time.Sleep(time.Millisecond * SLEEPTIME)
			fmt.Printf("Worker %d received job - %s\n", w.id, s)
		}
	}
}

func CreatePool() *Pool {
	pool := &Pool{jobsChan: make(chan string),
		Add:    make(chan struct{}),
		Delete: make(chan struct{}),
	}

	go pool.Manage()
	return pool
}

func (p *Pool) Manage() {
	for {
		select {
		case <-p.Add:
			p.AddWorker()
		case <-p.Delete:
			p.DeleteWorker()
		}
	}
}

func (p *Pool) AddWorker() {
	worker := Worker{id: len(p.workers) + 1,
		pool: p,
		done: make(chan struct{}),
	}
	p.workers = append(p.workers, &worker)
	p.wg.Add(1)

	go func() {
		defer p.wg.Done()
		worker.Start()
	}()
	fmt.Printf("worker %d added\n", worker.id)
}

func (p *Pool) DeleteWorker() {
	if len(p.workers) == 0 {
		fmt.Println("No workers to delete.")
		return
	}
	worker := p.workers[len(p.workers)-1]
	close(worker.done)
	p.workers = p.workers[:len(p.workers)-1]
	fmt.Printf("Worker %d deleted\n", worker.id)
}

func (p *Pool) DoWork(s string) {
	defer p.sendWg.Done()
	p.jobsChan <- s
}

func (p *Pool) Shutdown() {
	p.sendWg.Wait()
	fmt.Println("Shutdown started")
	close(p.jobsChan)
	for _, worker := range p.workers {
		close(worker.done)
		fmt.Printf("Worker %d deleted\n", worker.id)
	}
	p.workers = []*Worker{}
	p.wg.Wait()
	fmt.Println("Shutdown ended successfully")
}

func main() {
	pool := CreatePool()

	pool.Add <- struct{}{}
	pool.Add <- struct{}{}

	go func() {
		pool.sendWg.Add(JOBS)
		for i := range JOBS {
			pool.DoWork(fmt.Sprintf("Job %d", i))
		}
	}()

	time.Sleep(time.Millisecond * 500)
	// добавление воркеров
	pool.Add <- struct{}{}
	pool.Add <- struct{}{}

	time.Sleep(time.Millisecond * 2000)
	// удаление воркеров
	pool.Delete <- struct{}{}

	// ожидание записи всех тасок и закрытие всех каналов.
	pool.Shutdown()
}
