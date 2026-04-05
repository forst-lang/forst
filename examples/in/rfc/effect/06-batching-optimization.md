# Batching and Optimization Patterns

## Approach

Implement Effect-TS style batching and optimization patterns in Forst, enabling efficient handling of external API calls, database operations, and other I/O operations through request batching and caching mechanisms.

## Request Batching System

### Request Types

```go
// Request represents a batched operation
type Request[A, E any] interface {
    // Equality for batching identical requests
    Equals(other Request[A, E]) bool
    // Hash for efficient grouping
    Hash() uint64
    // Execute the request
    Execute() (A, E, bool)
}

// BatchedRequest groups multiple requests
type BatchedRequest[A, E any] struct {
    requests []Request[A, E]
    batchSize int
    timeout time.Duration
}

// Request constructor
func NewRequest[A, E any](
    id string,
    execute func() (A, E, bool),
) Request[A, E] {
    return &request[A, E]{
        id: id,
        execute: execute,
    }
}

type request[A, E any] struct {
    id string
    execute func() (A, E, bool)
}

func (r *request[A, E]) Equals(other Request[A, E]) bool {
    if otherReq, ok := other.(*request[A, E]); ok {
        return r.id == otherReq.id
    }
    return false
}

func (r *request[A, E]) Hash() uint64 {
    return hashString(r.id)
}

func (r *request[A, E]) Execute() (A, E, bool) {
    return r.execute()
}
```

### Batch Resolvers

```go
// BatchResolver handles multiple requests efficiently
type BatchResolver[A, E any] interface {
    // Resolve multiple requests in a single operation
    Resolve(requests []Request[A, E]) ([]A, E, bool)
    // Get batch size limit
    GetBatchSize() int
    // Get timeout for batching
    GetTimeout() time.Duration
}

// Database batch resolver
type DatabaseBatchResolver struct {
    db Database
    batchSize int
    timeout time.Duration
}

func NewDatabaseBatchResolver(db Database) *DatabaseBatchResolver {
    return &DatabaseBatchResolver{
        db: db,
        batchSize: 100,
        timeout: 100 * time.Millisecond,
    }
}

func (r *DatabaseBatchResolver) Resolve(requests []Request[User, DatabaseError]) ([]User, DatabaseError, bool) {
    // Extract user IDs from requests
    ids := make([]int, len(requests))
    for i, req := range requests {
        if userReq, ok := req.(*GetUserRequest); ok {
            ids[i] = userReq.userId
        }
    }

    // Batch fetch users
    users, err := r.db.GetUsersBatch(ids)
    if err != nil {
        return nil, NewDatabaseError(err.Error(), 500), false
    }

    return users, nil, true
}

func (r *DatabaseBatchResolver) GetBatchSize() int {
    return r.batchSize
}

func (r *DatabaseBatchResolver) GetTimeout() time.Duration {
    return r.timeout
}
```

### Batching Effect

```go
// BatchingEffect provides automatic request batching
type BatchingEffect[A, E any] struct {
    resolver BatchResolver[A, E]
    requests chan Request[A, E]
    results  map[string]chan Result[A, E]
    mu       sync.RWMutex
}

func NewBatchingEffect[A, E any](resolver BatchResolver[A, E]) *BatchingEffect[A, E] {
    be := &BatchingEffect[A, E]{
        resolver: resolver,
        requests: make(chan Request[A, E], 1000),
        results:  make(map[string]chan Result[A, E]),
    }

    // Start batching goroutine
    go be.batchLoop()

    return be
}

func (be *BatchingEffect[A, E]) batchLoop() {
    ticker := time.NewTicker(be.resolver.GetTimeout())
    defer ticker.Stop()

    var batch []Request[A, E]

    for {
        select {
        case req := <-be.requests:
            batch = append(batch, req)

            // Process batch if it's full
            if len(batch) >= be.resolver.GetBatchSize() {
                be.processBatch(batch)
                batch = nil
            }

        case <-ticker.C:
            // Process batch on timeout
            if len(batch) > 0 {
                be.processBatch(batch)
                batch = nil
            }
        }
    }
}

func (be *BatchingEffect[A, E]) processBatch(batch []Request[A, E]) {
    // Group requests by type and deduplicate
    grouped := be.groupRequests(batch)

    // Resolve each group
    for _, group := range grouped {
        go be.resolveGroup(group)
    }
}

func (be *BatchingEffect[A, E]) groupRequests(batch []Request[A, E]) map[string][]Request[A, E] {
    groups := make(map[string][]Request[A, E])

    for _, req := range batch {
        // Group by request type
        reqType := reflect.TypeOf(req).String()
        groups[reqType] = append(groups[reqType], req)
    }

    return groups
}

func (be *BatchingEffect[A, E]) resolveGroup(requests []Request[A, E]) {
    // Resolve batch
    results, err, success := be.resolver.Resolve(requests)

    if !success {
        // Send error to all requests
        for _, req := range requests {
            be.sendResult(req, Result[A, E]{Error: err, Success: false})
        }
        return
    }

    // Send results to individual requests
    for i, req := range requests {
        if i < len(results) {
            be.sendResult(req, Result[A, E]{Value: results[i], Success: true})
        } else {
            be.sendResult(req, Result[A, E]{Error: err, Success: false})
        }
    }
}

func (be *BatchingEffect[A, E]) sendResult(req Request[A, E], result Result[A, E]) {
    be.mu.Lock()
    defer be.mu.Unlock()

    if ch, exists := be.results[req.Hash()]; exists {
        select {
        case ch <- result:
        default:
        }
    }
}
```

## Caching System

### Cache Interface

```go
// Cache provides caching for Effect results
type Cache[A, E any] interface {
    Get(key string) (A, bool)
    Set(key string, value A, ttl time.Duration)
    Delete(key string)
    Clear()
}

// Memory cache implementation
type MemoryCache[A, E any] struct {
    data map[string]CacheEntry[A]
    mu   sync.RWMutex
}

type CacheEntry[A any] struct {
    value A
    expiry time.Time
}

func NewMemoryCache[A, E any]() *MemoryCache[A, E] {
    return &MemoryCache[A, E]{
        data: make(map[string]CacheEntry[A]),
    }
}

func (c *MemoryCache[A, E]) Get(key string) (A, bool) {
    c.mu.RLock()
    defer c.mu.RUnlock()

    entry, exists := c.data[key]
    if !exists || time.Now().After(entry.expiry) {
        var zero A
        return zero, false
    }

    return entry.value, true
}

func (c *MemoryCache[A, E]) Set(key string, value A, ttl time.Duration) {
    c.mu.Lock()
    defer c.mu.Unlock()

    c.data[key] = CacheEntry[A]{
        value: value,
        expiry: time.Now().Add(ttl),
    }
}

func (c *MemoryCache[A, E]) Delete(key string) {
    c.mu.Lock()
    defer c.mu.Unlock()
    delete(c.data, key)
}

func (c *MemoryCache[A, E]) Clear() {
    c.mu.Lock()
    defer c.mu.Unlock()
    c.data = make(map[string]CacheEntry[A])
}
```

### Cached Effect

```go
// CachedEffect provides caching for Effect results
type CachedEffect[A, E any] struct {
    cache Cache[A, E]
    effect Effect[A, E]
    keyGen func(A) string
    ttl time.Duration
}

func NewCachedEffect[A, E any](
    cache Cache[A, E],
    effect Effect[A, E],
    keyGen func(A) string,
    ttl time.Duration,
) *CachedEffect[A, E] {
    return &CachedEffect[A, E]{
        cache: cache,
        effect: effect,
        keyGen: keyGen,
        ttl: ttl,
    }
}

func (ce *CachedEffect[A, E]) Run() (A, E, bool) {
    // Try to get from cache first
    if value, found := ce.cache.Get(ce.keyGen(value)); found {
        return value, nil, true
    }

    // Execute effect
    value, err, success := ce.effect.Run()
    if !success {
        return value, err, false
    }

    // Cache result
    ce.cache.Set(ce.keyGen(value), value, ce.ttl)

    return value, nil, true
}
```

## Forst Integration

### Batching Effect Types

```go
// Forst code that uses batching
import effect "github.com/forst-lang/effect-runtime"

// Batching effect type
type BatchingEffect[A, E] = effect.BatchingEffect[A, E]

// Request types
type GetUserRequest struct {
    userId: Int
    requestId: String
}

func (r GetUserRequest) Equals(other effect.Request[User, DatabaseError]) Bool {
    if otherReq, ok := other.(GetUserRequest); ok {
        return r.userId == otherReq.userId
    }
    return false
}

func (r GetUserRequest) Hash() UInt64 {
    return hashInt(int(r.userId))
}

func (r GetUserRequest) Execute() (User, DatabaseError, Bool) {
    return getUserById(r.userId)
}

// Batch resolver
func NewUserBatchResolver(db Database) effect.BatchResolver[User, DatabaseError] {
    return &UserBatchResolver{db: db}
}

type UserBatchResolver struct {
    db Database
}

func (r UserBatchResolver) Resolve(requests []effect.Request[User, DatabaseError]) ([]User, DatabaseError, Bool) {
    // Extract user IDs
    ids := requests.Map(func(req effect.Request[User, DatabaseError]) Int {
        if userReq, ok := req.(GetUserRequest); ok {
            return userReq.userId
        }
        return -1
    }).Filter(func(id Int) Bool { return id >= 0 })

    // Batch fetch
    users, err := r.db.GetUsersBatch(ids)
    if err != nil {
        return List[User]{}, NewDatabaseError(err.Error(), 500), false
    }

    return users, nil, true
}

func (r UserBatchResolver) GetBatchSize() Int {
    return 100
}

func (r UserBatchResolver) GetTimeout() Duration {
    return 100 * time.Millisecond
}
```

### Usage Examples

```go
// Create batching effect
func createUserBatchingEffect(db Database) BatchingEffect[User, DatabaseError] {
    resolver := NewUserBatchResolver(db)
    return effect.NewBatchingEffect(resolver)
}

// Get user with batching
func getUserByIdBatched(id: Int, batchingEffect BatchingEffect[User, DatabaseError]) Effect[User, DatabaseError] {
    return effect.Async(func() (User, DatabaseError, Bool) {
        req := GetUserRequest{
            userId: id,
            requestId: generateId(),
        }

        // Submit request to batching effect
        result := batchingEffect.Submit(req)

        // Wait for result
        return result.Await()
    })
}

// Cached effect
func createCachedUserEffect(userId: Int, db Database) Effect[User, DatabaseError] {
    cache := effect.NewMemoryCache[User, DatabaseError]()
    effect := getUserById(userId)

    return effect.NewCachedEffect(
        cache: cache,
        effect: effect,
        keyGen: func(user User) String {
            return "user:" + user.Id.ToString()
        },
        ttl: 5 * time.Minute,
    )
}

// Composed batching and caching
func getUserWithBatchingAndCaching(id: Int, db Database) Effect[User, DatabaseError] {
    // Create batching effect
    batchingEffect := createUserBatchingEffect(db)

    // Create cached effect
    cachedEffect := createCachedUserEffect(id, db)

    // Combine batching and caching
    return effect.Async(func() (User, DatabaseError, Bool) {
        // Try cache first
        if user, found := cachedEffect.GetFromCache(); found {
            return user, nil, true
        }

        // Use batching if not in cache
        return getUserByIdBatched(id, batchingEffect).Run()
    })
}
```

### Advanced Batching Patterns

```go
// Request deduplication
func deduplicateRequests[A, E any](requests []Request[A, E]) []Request[A, E] {
    seen := make(map[uint64]bool)
    unique := make([]Request[A, E], 0, len(requests))

    for _, req := range requests {
        hash := req.Hash()
        if !seen[hash] {
            seen[hash] = true
            unique = append(unique, req)
        }
    }

    return unique
}

// Request prioritization
func prioritizeRequests[A, E any](requests []Request[A, E]) []Request[A, E] {
    // Sort by priority (implement priority logic)
    sort.Slice(requests, func(i, j int) bool {
        return requests[i].GetPriority() > requests[j].GetPriority()
    })

    return requests
}

// Adaptive batching
func adaptiveBatching[A, E any](resolver BatchResolver[A, E]) BatchResolver[A, E] {
    return &AdaptiveBatchResolver[A, E]{
        resolver: resolver,
        currentBatchSize: resolver.GetBatchSize(),
        successRate: 1.0,
    }
}

type AdaptiveBatchResolver[A, E any] struct {
    resolver BatchResolver[A, E]
    currentBatchSize int
    successRate float64
    mu sync.RWMutex
}

func (r *AdaptiveBatchResolver[A, E]) Resolve(requests []Request[A, E]) ([]A, E, bool) {
    // Adjust batch size based on success rate
    r.mu.Lock()
    if r.successRate > 0.9 {
        r.currentBatchSize = min(r.currentBatchSize*2, 1000)
    } else if r.successRate < 0.7 {
        r.currentBatchSize = max(r.currentBatchSize/2, 1)
    }
    r.mu.Unlock()

    // Resolve with current batch size
    results, err, success := r.resolver.Resolve(requests[:min(len(requests), r.currentBatchSize)])

    // Update success rate
    r.mu.Lock()
    if success {
        r.successRate = r.successRate*0.9 + 0.1
    } else {
        r.successRate = r.successRate * 0.9
    }
    r.mu.Unlock()

    return results, err, success
}
```

## Go Code Generation

### Generated Go Code

```go
// Batching effect generates to:
func createUserBatchingEffect(db Database) *effect.BatchingEffect[User, *effect.DatabaseError] {
    resolver := &UserBatchResolver{db: db}
    return effect.NewBatchingEffect(resolver)
}

func getUserByIdBatched(id int, batchingEffect *effect.BatchingEffect[User, *effect.DatabaseError]) effect.Effect[User, *effect.DatabaseError] {
    return effect.Async(func() (User, *effect.DatabaseError, bool) {
        req := &GetUserRequest{
            UserId: id,
            RequestId: generateId(),
        }

        result := batchingEffect.Submit(req)
        return result.Await()
    })
}

// Caching effect generates to:
func createCachedUserEffect(userId int, db Database) effect.Effect[User, *effect.DatabaseError] {
    cache := effect.NewMemoryCache[User, *effect.DatabaseError]()
    effect := getUserById(userId)

    return effect.NewCachedEffect(
        cache,
        effect,
        func(user User) string {
            return "user:" + strconv.Itoa(user.Id)
        },
        5*time.Minute,
    )
}
```

## TypeScript Generation

### Generated TypeScript

```typescript
// Batching types generate to:
interface BatchingEffect<A, E> {
  submit(request: Request<A, E>): Effect<A, E>;
  submitBatch(requests: Request<A, E>[]): Effect<A[], E>;
}

interface Request<A, E> {
  equals(other: Request<A, E>): boolean;
  hash(): number;
  execute(): { value: A; error: E; success: boolean };
}

interface BatchResolver<A, E> {
  resolve(requests: Request<A, E>[]): {
    values: A[];
    error: E;
    success: boolean;
  };
  getBatchSize(): number;
  getTimeout(): number;
}

// Caching types generate to:
interface Cache<A, E> {
  get(key: string): A | undefined;
  set(key: string, value: A, ttl: number): void;
  delete(key: string): void;
  clear(): void;
}

interface CachedEffect<A, E> {
  getFromCache(): A | undefined;
  invalidate(key: string): void;
  run(): Effect<A, E>;
}

// API functions
function createUserBatchingEffect(
  db: Database
): BatchingEffect<User, DatabaseError>;
function getUserByIdBatched(
  id: number,
  batchingEffect: BatchingEffect<User, DatabaseError>
): Effect<User, DatabaseError>;
function createCachedUserEffect(
  userId: number,
  db: Database
): Effect<User, DatabaseError>;
```

## Advantages

- **Performance**: Significant reduction in API calls through batching
- **Efficiency**: Automatic request deduplication and optimization
- **Caching**: Built-in caching for frequently accessed data
- **Adaptive**: Automatic adjustment of batch sizes based on performance
- **Type safety**: Full compile-time type checking for batching operations

## Challenges

- **Complexity**: Batching system adds significant complexity
- **Memory usage**: Caching and batching require additional memory
- **Debugging**: Batching can make debugging more difficult
- **Error handling**: Need to handle partial batch failures
- **Consistency**: Caching can lead to consistency issues

## Implementation Strategy

1. **Phase 1**: Basic request batching system
2. **Phase 2**: Request deduplication and grouping
3. **Phase 3**: Caching system and cache invalidation
4. **Phase 4**: Adaptive batching and optimization
5. **Phase 5**: Advanced patterns and monitoring
6. **Phase 6**: TypeScript generation and tooling
