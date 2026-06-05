import { useState, useEffect, useRef } from 'react';
import type { Tweet, AISummary, LoadTestResult } from './types';

const API_BASE = '/api';

function App() {
  const [tweets, setTweets] = useState<Tweet[]>([]);
  const [newTweet, setNewTweet] = useState('');
  const [loading, setLoading] = useState(true);
  const [aiSummary, setAiSummary] = useState<AISummary | null>(null);
  const [loadTest, setLoadTest] = useState<LoadTestResult | null>(null);
  const [activeTab, setActiveTab] = useState<'home' | 'ai' | 'stats'>('home');
  const wsRef = useRef<WebSocket | null>(null);

  useEffect(() => {
    fetchTimeline();
    connectWebSocket();
    return () => {
      wsRef.current?.close();
    };
  }, []);

  const fetchTimeline = async () => {
    try {
      const res = await fetch(`${API_BASE}/timeline`);
      const data = await res.json();
      setTweets(data.tweets);
    } catch (err) {
      console.error('Failed to fetch timeline:', err);
    } finally {
      setLoading(false);
    }
  };

  const connectWebSocket = () => {
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    const ws = new WebSocket(`${protocol}//${window.location.host}/ws`);

    ws.onmessage = (event) => {
      const data = JSON.parse(event.data);
      if (data.type === 'new_tweet') {
        setTweets(prev => [data.tweet, ...prev]);
      }
    };

    ws.onerror = () => {
      console.log('WebSocket connection failed (expected if server not running)');
    };

    wsRef.current = ws;
  };

  const postTweet = async () => {
    if (!newTweet.trim() || newTweet.length > 280) return;

    try {
      const res = await fetch(`${API_BASE}/tweet`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ content: newTweet }),
      });
      const tweet = await res.json();
      setTweets(prev => [tweet, ...prev]);
      setNewTweet('');
    } catch (err) {
      console.error('Failed to post tweet:', err);
    }
  };

  const likeTweet = async (id: number) => {
    try {
      await fetch(`${API_BASE}/tweet/${id}/like`, { method: 'POST' });
      setTweets(prev => prev.map(t =>
        t.id === id ? { ...t, like_count: t.like_count + 1 } : t
      ));
    } catch (err) {
      console.error('Failed to like:', err);
    }
  };

  const retweet = async (id: number) => {
    try {
      await fetch(`${API_BASE}/tweet/${id}/retweet`, { method: 'POST' });
      setTweets(prev => prev.map(t =>
        t.id === id ? { ...t, retweet_count: t.retweet_count + 1 } : t
      ));
    } catch (err) {
      console.error('Failed to retweet:', err);
    }
  };

  const getAISummary = async () => {
    try {
      const res = await fetch(`${API_BASE}/ai/summarize?hours=24`);
      const data = await res.json();
      setAiSummary(data);
    } catch (err) {
      console.error('Failed to get AI summary:', err);
    }
  };

  const runLoadTest = async () => {
    try {
      const res = await fetch(`${API_BASE}/loadtest`);
      const data = await res.json();
      setLoadTest(data);
    } catch (err) {
      console.error('Failed to run load test:', err);
    }
  };

  const formatTime = (dateStr: string) => {
    const date = new Date(dateStr);
    const now = new Date();
    const diff = Math.floor((now.getTime() - date.getTime()) / 1000);

    if (diff < 60) return `${diff}s`;
    if (diff < 3600) return `${Math.floor(diff / 60)}m`;
    if (diff < 86400) return `${Math.floor(diff / 3600)}h`;
    return `${Math.floor(diff / 86400)}d`;
  };

  return (
    <div className="min-h-screen bg-black text-white">
      <div className="max-w-2xl mx-auto border-x border-gray-800 min-h-screen flex flex-col">
        {/* Header */}
        <header className="sticky top-0 z-10 bg-black/80 backdrop-blur-md border-b border-gray-800 px-4 py-3">
          <div className="flex items-center justify-between">
            <h1 className="text-xl font-bold">Mini-X</h1>
            <div className="flex gap-2">
              <button
                onClick={() => setActiveTab('home')}
                className={`px-4 py-2 rounded-full text-sm font-medium transition-colors ${
                  activeTab === 'home'
                    ? 'bg-primary text-white'
                    : 'text-gray-400 hover:bg-gray-800'
                }`}
              >
                Home
              </button>
              <button
                onClick={() => {
                  setActiveTab('ai');
                  getAISummary();
                }}
                className={`px-4 py-2 rounded-full text-sm font-medium transition-colors ${
                  activeTab === 'ai'
                    ? 'bg-primary text-white'
                    : 'text-gray-400 hover:bg-gray-800'
                }`}
              >
                AI
              </button>
              <button
                onClick={() => {
                  setActiveTab('stats');
                  runLoadTest();
                }}
                className={`px-4 py-2 rounded-full text-sm font-medium transition-colors ${
                  activeTab === 'stats'
                    ? 'bg-primary text-white'
                    : 'text-gray-400 hover:bg-gray-800'
                }`}
              >
                Stats
              </button>
            </div>
          </div>
        </header>

        {activeTab === 'home' && (
          <>
            {/* Compose */}
            <div className="border-b border-gray-800 p-4">
              <div className="flex gap-3">
                <div className="w-10 h-10 rounded-full bg-gradient-to-br from-primary to-purple-500 flex items-center justify-center text-sm font-bold">
                  D
                </div>
                <div className="flex-1">
                  <textarea
                    value={newTweet}
                    onChange={(e) => setNewTweet(e.target.value)}
                    placeholder="What's happening?"
                    className="w-full bg-transparent text-lg resize-none outline-none placeholder-gray-500 min-h-[60px]"
                    rows={2}
                    maxLength={280}
                  />
                  <div className="flex items-center justify-between mt-2">
                    <div className="flex gap-2 text-primary">
                      {/* Icons */}
                      <button className="p-2 rounded-full hover:bg-blue-900/30 transition-colors">
                        <svg className="w-5 h-5" fill="currentColor" viewBox="0 0 24 24">
                          <path d="M3 5.5C3 4.119 4.119 3 5.5 3h13C19.881 3 21 4.119 21 5.5v13c0 1.381-1.119 2.5-2.5 2.5h-13A2.5 2.5 0 003 18.5v-13zM5.5 5c-.276 0-.5.224-.5.5v9.086l3-3 3 3 5-5 3 3V5.5c0-.276-.224-.5-.5-.5h-13zM19 15.414l-3-3-5 5-3-3-3 3V18.5c0 .276.224.5.5.5h13a.5.5 0 00.5-.5v-3.086z"/>
                        </svg>
                      </button>
                      <button className="p-2 rounded-full hover:bg-blue-900/30 transition-colors">
                        <svg className="w-5 h-5" fill="currentColor" viewBox="0 0 24 24">
                          <path d="M3 5.5C3 4.119 4.12 3 5.5 3h13C19.88 3 21 4.119 21 5.5v13c0 1.381-1.12 2.5-2.5 2.5h-13C4.12 21 3 19.881 3 18.5v-13zM5.5 5c-.28 0-.5.224-.5.5v13c0 .276.22.5.5.5h13c.28 0 .5-.224.5-.5v-13c0-.276-.22-.5-.5-.5h-13zM18 10.711V9.25h-3.74v5.5h1.44v-1.719h1.7V11.57h-1.7v-.859H18zM11.79 9.25h1.44v5.5h-1.44v-5.5zm-3.07 1.375c.34 0 .77.172 1.02.43l1.03-.86c-.51-.601-1.28-.945-2.05-.945C7.19 9.25 6 10.453 6 12s1.19 2.75 2.72 2.75c.85 0 1.54-.344 2.05-.945v-2.149H8.38v1.032H9.4v.515c-.17.086-.42.172-.68.172-.76 0-1.36-.602-1.36-1.375 0-.688.6-1.375 1.36-1.375z"/>
                        </svg>
                      </button>
                      <button className="p-2 rounded-full hover:bg-blue-900/30 transition-colors">
                        <svg className="w-5 h-5" fill="currentColor" viewBox="0 0 24 24">
                          <path d="M6 3V2H2v1H1v6.5l1.5 1.5H2v3h1v3h3v-3h1v3h3v-3h1v-3h1l1.5-1.5V2h-4v1H6zm6 4H6V5h6v2zm6 3h-3v3h-2v-3H9V7h3V4h2v3h3v3z"/>
                        </svg>
                      </button>
                    </div>
                    <div className="flex items-center gap-3">
                      <span className={`text-sm ${newTweet.length > 260 ? 'text-orange-500' : 'text-gray-500'}`}>
                        {280 - newTweet.length}
                      </span>
                      <button
                        onClick={postTweet}
                        disabled={!newTweet.trim() || newTweet.length > 280}
                        className="px-4 py-2 bg-primary text-white rounded-full font-medium disabled:opacity-50 disabled:cursor-not-allowed hover:bg-blue-500 transition-colors"
                      >
                        Post
                      </button>
                    </div>
                  </div>
                </div>
              </div>
            </div>

            {/* Timeline */}
            <div className="flex-1">
              {loading ? (
                <div className="flex items-center justify-center h-40">
                  <div className="w-8 h-8 border-2 border-primary border-t-transparent rounded-full animate-spin"></div>
                </div>
              ) : (
                tweets.map((tweet) => (
                  <TweetCard
                    key={tweet.id}
                    tweet={tweet}
                    onLike={likeTweet}
                    onRetweet={retweet}
                    formatTime={formatTime}
                  />
                ))
              )}
            </div>
          </>
        )}

        {activeTab === 'ai' && (
          <div className="p-6">
            <h2 className="text-2xl font-bold mb-4">AI Timeline Summary</h2>
            {aiSummary ? (
              <div className="space-y-6">
                <div className="bg-gray-900 rounded-xl p-6">
                  <h3 className="text-lg font-semibold mb-3 text-primary">Summary</h3>
                  <p className="text-gray-300 leading-relaxed">{aiSummary.summary}</p>
                </div>
                <div className="bg-gray-900 rounded-xl p-6">
                  <h3 className="text-lg font-semibold mb-3 text-primary">Key Topics</h3>
                  <div className="flex flex-wrap gap-2">
                    {aiSummary.topics.map((topic, i) => (
                      <span key={i} className="px-3 py-1 bg-gray-800 rounded-full text-sm">
                        #{topic}
                      </span>
                    ))}
                  </div>
                </div>
                <div className="grid grid-cols-2 gap-4">
                  <div className="bg-gray-900 rounded-xl p-4 text-center">
                    <div className="text-3xl font-bold text-primary">{aiSummary.tweet_count}</div>
                    <div className="text-gray-400 text-sm">Tweets Analyzed</div>
                  </div>
                  <div className="bg-gray-900 rounded-xl p-4 text-center">
                    <div className="text-3xl font-bold text-primary">{aiSummary.time_range}h</div>
                    <div className="text-gray-400 text-sm">Time Range</div>
                  </div>
                </div>
              </div>
            ) : (
              <div className="flex items-center justify-center h-40">
                <div className="w-8 h-8 border-2 border-primary border-t-transparent rounded-full animate-spin"></div>
              </div>
            )}
          </div>
        )}

        {activeTab === 'stats' && (
          <div className="p-6">
            <h2 className="text-2xl font-bold mb-4">Performance Stats</h2>
            {loadTest ? (
              <div className="space-y-6">
                <div className="bg-gray-900 rounded-xl p-6">
                  <h3 className="text-lg font-semibold mb-4 text-primary">Load Test Results</h3>
                  <div className="space-y-4">
                    <div className="flex justify-between items-center py-2 border-b border-gray-800">
                      <span className="text-gray-400">Simulated Users</span>
                      <span className="font-mono font-bold">{loadTest.users.toLocaleString()}</span>
                    </div>
                    <div className="flex justify-between items-center py-2 border-b border-gray-800">
                      <span className="text-gray-400">Total Tweets</span>
                      <span className="font-mono font-bold">{loadTest.tweets.toLocaleString()}</span>
                    </div>
                    <div className="flex justify-between items-center py-2 border-b border-gray-800">
                      <span className="text-gray-400">p95 Timeline Latency</span>
                      <span className="font-mono font-bold text-green-400">{loadTest.p95_timeline_ms.toFixed(2)}ms</span>
                    </div>
                    <div className="flex justify-between items-center py-2 border-b border-gray-800">
                      <span className="text-gray-400">p99 Post Latency</span>
                      <span className="font-mono font-bold text-yellow-400">{loadTest.p99_post_ms.toFixed(2)}ms</span>
                    </div>
                    <div className="flex justify-between items-center py-2">
                      <span className="text-gray-400">WebSocket Fanout</span>
                      <span className="font-mono font-bold text-blue-400">{loadTest.fanout_delay_ms}ms</span>
                    </div>
                  </div>
                </div>
                <button
                  onClick={runLoadTest}
                  className="w-full py-3 bg-gray-800 rounded-lg font-medium hover:bg-gray-700 transition-colors"
                >
                  Run Load Test Again
                </button>
              </div>
            ) : (
              <div className="flex flex-col items-center justify-center h-40 gap-4">
                <div className="w-8 h-8 border-2 border-primary border-t-transparent rounded-full animate-spin"></div>
                <p className="text-gray-400">Running load test...</p>
              </div>
            )}
          </div>
        )}
      </div>
    </div>
  );
}

function TweetCard({
  tweet,
  onLike,
  onRetweet,
  formatTime,
}: {
  tweet: Tweet;
  onLike: (id: number) => void;
  onRetweet: (id: number) => void;
  formatTime: (date: string) => string;
}) {
  return (
    <article className="border-b border-gray-800 p-4 hover:bg-gray-900/50 transition-colors">
      <div className="flex gap-3">
        <div className="w-10 h-10 rounded-full bg-gradient-to-br from-gray-500 to-gray-700 flex items-center justify-center text-sm font-bold flex-shrink-0">
          {tweet.user.display_name.charAt(0).toUpperCase()}
        </div>
        <div className="flex-1 min-w-0">
          <div className="flex items-center gap-2 flex-wrap">
            <span className="font-bold hover:underline cursor-pointer">
              {tweet.user.display_name}
            </span>
            <span className="text-gray-500">@{tweet.user.username}</span>
            {tweet.user.is_celebrity && (
              <span className="px-1.5 py-0.5 bg-primary/20 text-primary text-xs rounded font-medium">
                ✓
              </span>
            )}
            <span className="text-gray-500">·</span>
            <span className="text-gray-500 hover:underline cursor-pointer">
              {formatTime(tweet.created_at)}
            </span>
          </div>
          <p className="mt-1 whitespace-pre-wrap break-words">{tweet.content}</p>
          <div className="flex gap-6 mt-3 text-gray-500">
            <button
              onClick={() => onRetweet(tweet.id)}
              className="flex items-center gap-2 hover:text-green-400 transition-colors group"
            >
              <svg className="w-5 h-5 group-hover:bg-green-400/20 p-1 rounded-full" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M19.5 12c0-1.232-.046-2.453-.138-3.662a4.006 4.006 0 00-3.7-3.7 48.678 48.678 0 00-7.324 0 4.006 4.006 0 00-3.7 3.7c-.017.22-.032.441-.046.662M19.5 12l3-3m-3 3l-3-3m-12 3c0 1.232.046 2.453.138 3.662a4.006 4.006 0 003.7 3.7 48.656 48.656 0 007.324 0 4.006 4.006 0 003.7-3.7c.017-.22.032-.441.046-.662M4.5 12l3 3m-3-3l-3 3" />
              </svg>
              <span className="text-sm">{tweet.retweet_count}</span>
            </button>
            <button
              onClick={() => onLike(tweet.id)}
              className="flex items-center gap-2 hover:text-red-400 transition-colors group"
            >
              <svg className="w-5 h-5 group-hover:bg-red-400/20 p-1 rounded-full" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M4.318 6.318a4.5 4.5 0 000 6.364L12 20.364l7.682-7.682a4.5 4.5 0 00-6.364-6.364L12 7.636l-1.318-1.318a4.5 4.5 0 00-6.364 0z" />
              </svg>
              <span className="text-sm">{tweet.like_count}</span>
            </button>
            <button className="flex items-center gap-2 hover:text-blue-400 transition-colors group">
              <svg className="w-5 h-5 group-hover:bg-blue-400/20 p-1 rounded-full" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M8 12h.01M12 12h.01M16 12h.01M21 12c0 4.418-4.03 8-9 8a9.863 9.863 0 01-4.255-.949L3 20l1.395-3.72C3.512 15.042 3 13.574 3 12c0-4.418 4.03-8 9-8s9 3.582 9 8z" />
              </svg>
              <span className="text-sm">{tweet.reply_count}</span>
            </button>
          </div>
        </div>
      </div>
    </article>
  );
}

export default App;
