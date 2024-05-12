package calendarqueue

type CalendarQueue[T any] struct {
	buckets    []*event[T]
	numBuckets int
	lastBucket int

	bucketWidth float64
	bucketTop   float64

	lastPrio float64

	size               int
	topResizeThreshold int
	botResizeThreshold int

	resizeEnabled bool
}

func New[T any]() *CalendarQueue[T] {
	q := CalendarQueue[T]{}
	q.localInit(2, 1.0, 0.0)
	q.resizeEnabled = true
	return &q
}

func (q *CalendarQueue[T]) Enqueue(entry *event[T]) {
	i := int(entry.priority/q.bucketWidth) % q.numBuckets

	if q.buckets[i] == nil || q.buckets[i].priority > entry.priority {
		entry.next = q.buckets[i]
		q.buckets[i] = entry
	} else {
		current := q.buckets[i]
		for current.next != nil {
			if current.priority < entry.priority {
				current = current.next
			} else {
				break
			}
		}
		entry.next = current.next
		current.next = entry
	}

	q.size++
	if q.size > q.topResizeThreshold {
		q.resize(2 * q.numBuckets)
	}
}

func (q *CalendarQueue[T]) Dequeue() *event[T] {
	if q.size == 0 {
		return nil
	}
	i := q.lastBucket
	for {
		curr := q.buckets[i]
		if curr != nil && curr.priority < q.bucketTop {
			q.buckets[i] = curr.next
			q.lastBucket = i
			q.lastPrio = curr.priority
			q.size--
			if q.size < q.botResizeThreshold {
				q.resize(q.numBuckets / 2)
			}
			return curr
		} else {
			i++
			if i == q.numBuckets {
				i = 0
			}
			q.bucketTop += q.bucketWidth

			if i == q.lastBucket {
				break
			}
		}

	}

	var minPrio float64
	for i, bucket := range q.buckets {
		if bucket != nil {
			q.lastBucket = i
			q.lastPrio = bucket.priority
			minPrio = bucket.priority
			break
		}
	}

	for i := q.lastBucket; i < q.numBuckets; i++ {
		bucket := q.buckets[i]
		if bucket != nil && bucket.priority < minPrio {
			q.lastBucket = i
			q.lastPrio = bucket.priority
			minPrio = bucket.priority
		}
	}

	n := q.lastPrio / q.bucketWidth
	q.bucketTop = (n + 1.5) * q.bucketWidth

	return q.Dequeue()
}

func (q *CalendarQueue[T]) localInit(numBuckets int, bucketWidth, startPrio float64) {
	q.buckets = make([]*event[T], numBuckets)
	q.bucketWidth = bucketWidth
	q.numBuckets = numBuckets
	q.size = 0
	q.lastPrio = startPrio
	n := startPrio / bucketWidth
	q.lastBucket = int(n) % numBuckets
	q.bucketTop = (n + 1.5) * bucketWidth
	q.botResizeThreshold = numBuckets/2 - 2
	q.topResizeThreshold = numBuckets * 2
}

func (q *CalendarQueue[T]) resize(numBuckets int) {
	if !q.resizeEnabled {
		return
	}

	bucketWidth := q.newWidth()
	oldBuckets := q.buckets
	oldNumBuckets := q.numBuckets
	q.localInit(numBuckets, bucketWidth, q.lastPrio)
	for i := oldNumBuckets - 1; i >= 0; i-- {
		curr := oldBuckets[i]
		for curr != nil {
			e := NewEvent(curr.Data, curr.priority)
			q.Enqueue(e)
			curr = curr.next
		}
	}
}

func (q *CalendarQueue[T]) newWidth() float64 {
	var numSamples int
	if q.size < 2.0 {
		return 1.0
	}
	if q.size <= 5 {
		numSamples = q.size
	} else {
		numSamples = 5 + q.size/10
	}
	if numSamples > 25 {
		numSamples = 25
	}

	cachedLastPrio := q.lastPrio
	cachedLastBucket := q.lastBucket
	cachedBucketTop := q.bucketTop
	sampledEvents := make([]*event[T], numSamples)
	sampledPrios := make([]float64, numSamples)

	q.resizeEnabled = false
	for i := 0; i < numSamples; i++ {
		e := q.Dequeue()
		sampledEvents[i] = e
		sampledPrios[i] = e.priority
	}

	for i := numSamples - 1; i >= 0; i-- {
		q.Enqueue(sampledEvents[i])
	}

	q.lastPrio = cachedLastPrio
	q.lastBucket = cachedLastBucket
	q.bucketTop = cachedBucketTop
	q.resizeEnabled = true

	separationsSum := 0.0
	for i := 1; i < numSamples; i++ {
		separationsSum += sampledPrios[i] - sampledPrios[i-1]
	}
	avgSeparation := separationsSum / float64(numSamples-1)

	separationsSum = 0
	count := 0
	for i := 1; i < numSamples; i++ {
		separation := sampledPrios[i] - sampledPrios[i-1]
		if separation < avgSeparation*2 {
			separationsSum += separation
			count++
		}
	}
	recalculatedAverage := separationsSum / float64(count)

	return recalculatedAverage * 3
}

type event[T any] struct {
	Data     T
	priority float64
	next     *event[T]
}

func NewEvent[T any](data T, priority float64) *event[T] {
	return &event[T]{
		Data:     data,
		priority: priority,
		next:     nil,
	}
}
