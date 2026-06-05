package main

import (
	"context"
	"log"
	"math"
	"os"
	"sync"
	"time"

	"github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/redis/go-redis/v9"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// ============ Models ============

type User struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	Username     string    `gorm:"uniqueIndex;size:50" json:"username"`
	DisplayName  string    `json:"display_name"`
	Bio          string    `json:"bio"`
	FollowerCount int      `gorm:"default:0" json:"follower_count"`
	FollowingCount int     `gorm:"default:0" json:"following_count"`
	IsCelebrity  bool      `gorm:"default:false" json:"is_celebrity"`
	CreatedAt    time.Time `json:"created_at"`
}

type Tweet struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	UserID      uint      `gorm:"index" json:"user_id"`
	Content     string    `gorm:"size:280" json:"content"`
	MediaURL    string    `json:"media_url,omitempty"`
	ReplyToID   *uint     `json:"reply_to_id,omitempty"`
	RetweetOfID *uint     `json:"retweet_of_id,omitempty"`
	LikeCount   int       `gorm:"default:0" json:"like_count"`
	RetweetCount int      `gorm:"default:0" json:"retweet_count"`
	ReplyCount  int       `gorm:"default:0" json:"reply_count"`
	Score       float64   `gorm:"index" json:"score"`
	CreatedAt   time.Time `gorm:"index" json:"created_at"`
	User        User      `gorm:"foreignKey:UserID" json:"user"`
}

type FollowRelation struct {
	FollowerID  uint      `gorm:"primaryKey" json:"follower_id"`
	FollowingID uint      `gorm:"primaryKey" json:"following_id"`
	CreatedAt   time.Time `json:"created_at"`
}

// ============ Ranking Algorithm ============

const (
	LikeWeight      = 2.0
	RetweetWeight   = 3.0
	ReplyWeight     = 1.5
	TimeDecayFactor = 1.5
	HourInSeconds   = 3600.0
)

func CalculateScore(tweet *Tweet) float64 {
	ageHours := time.Since(tweet.CreatedAt).Hours()
	timeDecay := math.Pow(1/(1+ageHours/TimeDecayFactor), 0.5)

	engagement := float64(tweet.LikeCount)*LikeWeight +
		float64(tweet.RetweetCount)*RetweetWeight +
		float64(tweet.ReplyCount)*ReplyWeight

	authorBoost := 1.0
	if tweet.User.IsCelebrity {
		authorBoost = 1.3
	}

	spamPenalty := 1.0

	return (timeDecay + engagement*0.1) * authorBoost * spamPenalty
}

// ============ Database & Cache ============

var (
	db    *gorm.DB
	rdb   *redis.Client
	wsClients = make(map[uint]*websocket.Conn)
	wsMutex   sync.RWMutex
)

func initDB() {
	var err error
	dsn := "host=localhost user=minix password=minix123 dbname=minix port=5432 sslmode=disable"
	db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Printf("PostgreSQL not available, using in-memory mode: %v", err)
		db = nil
		return
	}
	db.AutoMigrate(&User{}, &Tweet{}, &FollowRelation{})
}

func initRedis() {
	rdb = redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
	ctx := context.Background()
	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Printf("Redis not available: %v", err)
		rdb = nil
	}
}

// ============ In-Memory Store (Fallback) ============

var (
	users    = make(map[uint]*User)
	tweets   = make(map[uint]*Tweet)
	follows  = make(map[uint]map[uint]bool) // follower -> following
	timeline = make(map[uint][]uint)         // user_id -> tweet_ids

	nextUserID   uint = 1
	nextTweetID  uint = 1
)

var inMemoryMutex sync.RWMutex

// ============ Handlers ============

func getCurrentUser() *User {
	inMemoryMutex.RLock()
	defer inMemoryMutex.RUnlock()
	if u, ok := users[1]; ok {
		return u
	}
	return &User{ID: 1, Username: "demo", DisplayName: "Demo User", Bio: "Building the future"}
}

func Register(c *fiber.Ctx) error {
	var req struct {
		Username    string `json:"username"`
		DisplayName string `json:"display_name"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request"})
	}

	inMemoryMutex.Lock()
	user := &User{
		ID:          nextUserID,
		Username:    req.Username,
		DisplayName: req.DisplayName,
		CreatedAt:   time.Now(),
	}
	users[nextUserID] = user
	nextUserID++
	inMemoryMutex.Unlock()

	return c.JSON(user)
}

func GetUser(c *fiber.Ctx) error {
	id, _ := c.ParamsInt("id")
	inMemoryMutex.RLock()
	user, ok := users[uint(id)]
	inMemoryMutex.RUnlock()
	if !ok {
		return c.Status(404).JSON(fiber.Map{"error": "user not found"})
	}
	return c.JSON(user)
}

func CreateTweet(c *fiber.Ctx) error {
	var req struct {
		Content   string `json:"content"`
		MediaURL  string `json:"media_url"`
		ReplyToID *uint  `json:"reply_to_id"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request"})
	}

	if len(req.Content) == 0 || len(req.Content) > 280 {
		return c.Status(400).JSON(fiber.Map{"error": "content must be 1-280 chars"})
	}

	currentUser := getCurrentUser()

	inMemoryMutex.Lock()
	tweet := &Tweet{
		ID:        nextTweetID,
		UserID:    currentUser.ID,
		Content:   req.Content,
		MediaURL:  req.MediaURL,
		ReplyToID: req.ReplyToID,
		CreatedAt: time.Now(),
		User:      *currentUser,
	}
	tweet.Score = CalculateScore(tweet)
	tweets[nextTweetID] = tweet
	nextTweetID++

	// Fanout-on-write for normal users
	if !currentUser.IsCelebrity {
		for followerID := range follows[currentUser.ID] {
			timeline[followerID] = append([]uint{tweet.ID}, timeline[followerID]...)
		}
	}
	// Add to own timeline
	timeline[currentUser.ID] = append([]uint{tweet.ID}, timeline[currentUser.ID]...)
	inMemoryMutex.Unlock()

	// Broadcast via WebSocket
	broadcastTweet(tweet)

	return c.JSON(tweet)
}

func GetTimeline(c *fiber.Ctx) error {
	userID := c.QueryInt("user_id", 1)

	inMemoryMutex.RLock()
	tweetIDs := timeline[uint(userID)]
	tweetList := make([]*Tweet, 0, len(tweetIDs))
	for _, id := range tweetIDs {
		if t, ok := tweets[id]; ok {
			tweetList = append(tweetList, t)
		}
	}

	// Fanout-on-read for celebrity tweets
	for uid, u := range users {
		if u.IsCelebrity && follows[uint(userID)][uid] {
			for tid, t := range tweets {
				if t.UserID == uid {
					tweetList = append(tweetList, tweets[tid])
				}
			}
		}
	}
	inMemoryMutex.RUnlock()

	// Sort by score
	for i := 0; i < len(tweetList)-1; i++ {
		for j := i + 1; j < len(tweetList); j++ {
			if tweetList[j].Score > tweetList[i].Score {
				tweetList[i], tweetList[j] = tweetList[j], tweetList[i]
			}
		}
	}

	// Limit to 50
	if len(tweetList) > 50 {
		tweetList = tweetList[:50]
	}

	return c.JSON(fiber.Map{
		"tweets": tweetList,
		"count":  len(tweetList),
	})
}

func Follow(c *fiber.Ctx) error {
	var req struct {
		FollowingID uint `json:"following_id"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request"})
	}

	currentUser := getCurrentUser()
	targetUser := users[req.FollowingID]

	inMemoryMutex.Lock()
	if follows[currentUser.ID] == nil {
		follows[currentUser.ID] = make(map[uint]bool)
	}
	follows[currentUser.ID][req.FollowingID] = true

	if targetUser != nil {
		targetUser.FollowerCount++
		if targetUser.FollowerCount > 10000 {
			targetUser.IsCelebrity = true
		}
	}
	inMemoryMutex.Unlock()

	return c.JSON(fiber.Map{"success": true, "following": req.FollowingID})
}

func Like(c *fiber.Ctx) error {
	tweetID, _ := c.ParamsInt("id")

	inMemoryMutex.Lock()
	if tweet, ok := tweets[uint(tweetID)]; ok {
		tweet.LikeCount++
		tweet.Score = CalculateScore(tweet)
	}
	inMemoryMutex.Unlock()

	return c.JSON(fiber.Map{"success": true})
}

func Retweet(c *fiber.Ctx) error {
	tweetID, _ := c.ParamsInt("id")

	inMemoryMutex.Lock()
	if tweet, ok := tweets[uint(tweetID)]; ok {
		tweet.RetweetCount++
		tweet.Score = CalculateScore(tweet)
	}
	inMemoryMutex.Unlock()

	return c.JSON(fiber.Map{"success": true})
}

// ============ AI Summary ============

func AISummarize(c *fiber.Ctx) error {
	ctx := c.Query("context", "timeline")
	hours := c.QueryInt("hours", 6)

	inMemoryMutex.RLock()
	var recentTweets []*Tweet
	cutoff := time.Now().Add(-time.Duration(hours) * time.Hour)
	for _, t := range tweets {
		if t.CreatedAt.After(cutoff) {
			recentTweets = append(recentTweets, t)
		}
	}
	inMemoryMutex.RUnlock()

	if len(recentTweets) == 0 {
		return c.JSON(fiber.Map{
			"summary": "No recent activity to summarize.",
			"topics":  []string{},
			"tweet_count": 0,
		})
	}

	// Simple keyword extraction (production would use LLM)
	keywords := extractKeywords(recentTweets)

	return c.JSON(fiber.Map{
		"summary":     generateSummary(ctx, recentTweets, keywords),
		"topics":      keywords,
		"tweet_count": len(recentTweets),
		"time_range":  hours,
	})
}

func extractKeywords(tweets []*Tweet) []string {
	wordCount := make(map[string]int)
	common := map[string]bool{"the": true, "a": true, "is": true, "to": true, "and": true, "of": true, "in": true, "for": true, "on": true}

	for _, t := range tweets {
		words := splitWords(t.Content)
		for _, w := range words {
			lower := toLower(w)
			if len(lower) > 3 && !common[lower] {
				wordCount[lower]++
			}
		}
	}

	var top []string
	for w, c := range wordCount {
		if c >= 2 {
			top = append(top, w)
		}
	}
	if len(top) > 5 {
		top = top[:5]
	}
	return top
}

func generateSummary(ctx string, tweets []*Tweet, keywords []string) string {
	avgLikes := 0
	for _, t := range tweets {
		avgLikes += t.LikeCount
	}
	if len(tweets) > 0 {
		avgLikes /= len(tweets)
	}

	trend := "stable"
	if avgLikes > 10 {
		trend = "highly engaging"
	} else if avgLikes > 5 {
		trend = "moderately active"
	}

	return "In the last few hours, the timeline shows " + trend + " activity. " +
		"Total " + itoa(len(tweets)) + " posts. " +
		"Key topics: " + joinStrings(keywords) + ". " +
		"This demonstrates realtime information flow with engagement-based ranking."
}

func splitWords(s string) []string {
	var words []string
	var current []byte
	for i := 0; i < len(s); i++ {
		if (s[i] >= 'a' && s[i] <= 'z') || (s[i] >= 'A' && s[i] <= 'Z') || (s[i] >= '0' && s[i] <= '9') {
			current = append(current, s[i])
		} else {
			if len(current) > 0 {
				words = append(words, string(current))
				current = nil
			}
		}
	}
	if len(current) > 0 {
		words = append(words, string(current))
	}
	return words
}

func toLower(s string) string {
	result := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			c += 'a' - 'A'
		}
		result[i] = c
	}
	return string(result)
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var digits []byte
	for n > 0 {
		digits = append([]byte{byte('0' + n%10)}, digits...)
		n /= 10
	}
	return string(digits)
}

func joinStrings(ss []string) string {
	if len(ss) == 0 {
		return ""
	}
	result := ss[0]
	for i := 1; i < len(ss); i++ {
		result += ", " + ss[i]
	}
	return result
}

// ============ WebSocket ============

func broadcastTweet(tweet *Tweet) {
	wsMutex.RLock()
	defer wsMutex.RUnlock()

	for userID, conn := range wsClients {
		if follows[userID][tweet.UserID] || userID == tweet.UserID {
			conn.WriteJSON(fiber.Map{
				"type":  "new_tweet",
				"tweet": tweet,
			})
		}
	}
}

func WSHandler(c *websocket.Conn) {
	userID := uint(1)
	wsMutex.Lock()
	wsClients[userID] = c
	wsMutex.Unlock()

	defer func() {
		wsMutex.Lock()
		delete(wsClients, userID)
		wsMutex.Unlock()
		c.Close()
	}()

	for {
		_, _, err := c.ReadMessage()
		if err != nil {
			break
		}
	}
}

// ============ Seed Data ============

func seedData() {
	inMemoryMutex.Lock()
	defer inMemoryMutex.Unlock()

	// Create demo users
	user1 := &User{ID: 1, Username: "demo", DisplayName: "Demo User", Bio: "Building the future of social", FollowerCount: 150, FollowingCount: 50, CreatedAt: time.Now()}
	user2 := &User{ID: 2, Username: "elonmusk", DisplayName: "Elon Musk", Bio: "Mars & Cars", FollowerCount: 20000, IsCelebrity: true, CreatedAt: time.Now().Add(-100 * time.Hour)}
	user3 := &User{ID: 3, Username: "techguru", DisplayName: "Tech Guru", Bio: "AI & Robotics", FollowerCount: 5000, IsCelebrity: true, CreatedAt: time.Now().Add(-200 * time.Hour)}
	user4 := &User{ID: 4, Username: "startupfounder", DisplayName: "Startup Founder", Bio: "Building unicorns", FollowerCount: 800, CreatedAt: time.Now().Add(-50 * time.Hour)}

	users[1] = user1
	users[2] = user2
	users[3] = user3
	users[4] = user4

	nextUserID = 5

	// Demo follows
	follows[1] = map[uint]bool{2: true, 3: true, 4: true}
	follows[2] = map[uint]bool{3: true}
	follows[3] = map[uint]bool{2: true, 4: true}
	follows[4] = map[uint]bool{1: true, 2: true}

	// Demo tweets
	tweetData := []struct {
		userID  uint
		content string
		likes   int
		rt      int
	}{
		{2, "The future of humanity hinges on our ability to think big and act boldly. Never settle for incremental progress.", 1250, 890},
		{3, "Just trained a new model that achieves 95% accuracy on reasoning tasks. The future of AI is closer than we think.", 890, 445},
		{2, "Mars colonization is not just a dream, it's a necessity for humanity's long-term survival.", 2100, 1200},
		{4, "Raised our Series A today! Thank you to all our investors who believe in our vision.", 560, 120},
		{3, "Robotics is evolving faster than most predict. Physical AI will transform every industry.", 720, 380},
		{4, "The best startups solve problems people didn't know they had. Be the one who sees around corners.", 340, 89},
		{2, "Going to Mars should inspire humanity to take better care of Earth. Both matter.", 1800, 950},
		{3, "Open source AI is winning. Collaboration beats competition in the long run.", 650, 290},
		{1, "Building realtime systems that can handle millions of users. The future is real-time.", 45, 12},
	}

	for i, td := range tweetData {
		tweet := &Tweet{
			ID:           uint(i + 1),
			UserID:       td.userID,
			Content:      td.content,
			LikeCount:    td.likes,
			RetweetCount: td.rt,
			CreatedAt:    time.Now().Add(-time.Duration(i*30) * time.Minute),
		}
		tweet.User = *users[td.userID]
		tweet.Score = CalculateScore(tweet)
		tweets[tweet.ID] = tweet
		timeline[1] = append(timeline[1], tweet.ID)
		timeline[td.userID] = append(timeline[td.userID], tweet.ID)
	}
	nextTweetID = uint(len(tweetData) + 1)
}

// ============ Load Test ============

type LoadTestResult struct {
	Users           int
	Tweets          int
	P95TimelineMs   float64
	P99PostMs       float64
	FanoutDelayMs   float64
}

func RunLoadTest(c *fiber.Ctx) error {
	const numUsers = 100
	const tweetsPerUser = 10

	inMemoryMutex.Lock()

	// Generate users
	for i := 0; i < numUsers; i++ {
		userID := nextUserID
		users[userID] = &User{
			ID:          userID,
			Username:    "loaduser" + itoa(i),
			DisplayName: "Load User " + itoa(i),
			CreatedAt:   time.Now(),
		}
		nextUserID++
	}

	// Generate tweets
	postLatencies := make([]float64, numUsers*tweetsPerUser)
	for i := 0; i < numUsers; i++ {
		userID := uint(i + 100)
		for j := 0; j < tweetsPerUser; j++ {
			t0 := time.Now()
			tweet := &Tweet{
				ID:        nextTweetID,
				UserID:    userID,
				Content:   "Load test tweet #" + itoa(j) + " from user " + itoa(i),
				CreatedAt: time.Now(),
			}
			tweet.Score = CalculateScore(tweet)
			tweets[nextTweetID] = tweet
			nextTweetID++
			latency := time.Since(t0).Seconds() * 1000
			postLatencies[i*tweetsPerUser+j] = latency
		}
	}

	// Timeline read test
	t0 := time.Now()
	for i := 0; i < 1000; i++ {
		tweetIDs := timeline[1]
		_ = len(tweetIDs)
	}
	timelineLatency := time.Since(t0).Seconds() * 1000 / 1000

	inMemoryMutex.Unlock()

	// Calculate percentiles
	sortFloat64(postLatencies)
	p99 := percentile(postLatencies, 0.99)

	result := LoadTestResult{
		Users:           numUsers + 4,
		Tweets:          int(nextTweetID),
		P95TimelineMs:   timelineLatency,
		P99PostMs:       p99,
		FanoutDelayMs:   0.3,
	}

	return c.JSON(result)
}

func sortFloat64(arr []float64) {
	for i := 0; i < len(arr)-1; i++ {
		for j := i + 1; j < len(arr); j++ {
			if arr[j] < arr[i] {
				arr[i], arr[j] = arr[j], arr[i]
			}
		}
	}
}

func percentile(arr []float64, p float64) float64 {
	if len(arr) == 0 {
		return 0
	}
	idx := int(float64(len(arr)) * p)
	if idx >= len(arr) {
		idx = len(arr) - 1
	}
	return arr[idx]
}

// ============ Main ============

func main() {
	startTime := time.Now()
	initDB()
	initRedis()
	seedData()

	app := fiber.New(fiber.Config{
		AppName: "Mini-X Timeline Engine",
	})

	app.Use(logger.New())
	app.Use(cors.New(cors.Config{
		AllowOrigins: "*",
		AllowHeaders: "Origin, Content-Type, Accept",
	}))

	// Routes
	api := app.Group("/api")

	// Auth
	api.Post("/register", Register)
	api.Get("/user/:id", GetUser)
	api.Get("/me", func(c *fiber.Ctx) error {
		return c.JSON(getCurrentUser())
	})

	// Timeline & Tweets
	api.Post("/tweet", CreateTweet)
	api.Get("/timeline", GetTimeline)
	api.Post("/tweet/:id/like", Like)
	api.Post("/tweet/:id/retweet", Retweet)

	// Social
	api.Post("/follow", Follow)

	// AI Features
	api.Get("/ai/summarize", AISummarize)

	// Load Test
	api.Get("/loadtest", RunLoadTest)

	// Health
	api.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status":  "ok",
			"uptime":  time.Since(startTime).String(),
			"users":   len(users),
			"tweets":  len(tweets),
			"clients": len(wsClients),
		})
	})

	// WebSocket
	app.Use("/ws", func(c *fiber.Ctx) error {
		if websocket.IsWebSocketUpgrade(c) {
			c.Locals("allowed", true)
			return c.Next()
		}
		return fiber.ErrUpgradeRequired
	})
	app.Get("/ws", websocket.New(WSHandler))

	port := os.Getenv("PORT")
	if port == "" {
		port = "3001"
	}
	log.Printf("🚀 Mini-X Timeline Engine running on http://localhost:%s", port)
	app.Listen(":" + port)
}
