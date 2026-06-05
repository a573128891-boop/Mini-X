export interface User {
  id: number;
  username: string;
  display_name: string;
  bio: string;
  follower_count: number;
  following_count: number;
  is_celebrity: boolean;
  created_at: string;
}

export interface Tweet {
  id: number;
  user_id: number;
  content: string;
  media_url?: string;
  reply_to_id?: number;
  retweet_of_id?: number;
  like_count: number;
  retweet_count: number;
  reply_count: number;
  score: number;
  created_at: string;
  user: User;
}

export interface TimelineResponse {
  tweets: Tweet[];
  count: number;
}

export interface AISummary {
  summary: string;
  topics: string[];
  tweet_count: number;
  time_range: number;
}

export interface LoadTestResult {
  users: number;
  tweets: number;
  p95_timeline_ms: number;
  p99_post_ms: number;
  fanout_delay_ms: number;
}
