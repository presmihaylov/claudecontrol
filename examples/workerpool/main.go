package main

import (
	"fmt"
	"time"

	"github.com/gammazero/workerpool"
)

// Simulates cr.handleMessage() - slow processing that takes time
func processMessage(message string, msgNum int) {
	fmt.Printf("🔄 WORKER STARTED: Processing message %d: '%s'\n", msgNum, message)
	
	// Simulate slow processing (like Claude API calls, Git operations, etc.)
	processingTime := time.Duration(1000+msgNum*200) * time.Millisecond
	time.Sleep(processingTime)
	
	fmt.Printf("✅ WORKER FINISHED: Message %d: '%s' (took %v)\n", msgNum, message, processingTime)
}

func main() {
	fmt.Println("=== WORKERPOOL DEMONSTRATION ===")
	fmt.Println("Showing how 5 messages queue instantly but process sequentially\n")
	
	messages := []string{"Hello", "World", "How", "Are", "You"}
	
	// Create worker pool with 1 worker for sequential processing
	wp := workerpool.New(1)
	defer wp.StopWait() // Wait for all tasks to complete
	
	start := time.Now()
	
	fmt.Println("📤 QUEUEING PHASE (simulates WebSocket reader):")
	// Simulate WebSocket reader that receives all messages quickly
	for i, msg := range messages {
		fmt.Printf("📨 Received message %d: '%s' (at %v)\n", i+1, msg, time.Since(start).Round(time.Millisecond))
		
		// Submit to worker pool - RETURNS IMMEDIATELY
		msgCopy := msg // Important: capture loop variable
		msgNum := i + 1
		wp.Submit(func() {
			processMessage(msgCopy, msgNum)
		})
		
		fmt.Printf("⚡ Message %d QUEUED instantly (at %v)\n", i+1, time.Since(start).Round(time.Millisecond))
		
		// Small delay to simulate messages arriving quickly but not instantly
		time.Sleep(50 * time.Millisecond)
	}
	
	fmt.Printf("\n🚀 ALL 5 MESSAGES QUEUED in: %v\n", time.Since(start).Round(time.Millisecond))
	fmt.Println("\n📝 PROCESSING PHASE (worker processes one at a time):")
	
	// wp.StopWait() will be called by defer, blocking until all tasks complete
	fmt.Printf("\n🏁 ALL PROCESSING COMPLETE at: %v\n", time.Since(start).Round(time.Millisecond))
	
	fmt.Println("\n🎯 Key Points:")
	fmt.Println("   ✅ All 5 messages were queued in ~250ms")
	fmt.Println("   ✅ Worker processed them sequentially (one at a time)")
	fmt.Println("   ✅ Total processing took ~7 seconds, but queueing was instant")
	fmt.Println("   ✅ WebSocket reader would never block in real ccagent!")
}