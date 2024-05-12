package calendarqueue

// TODO: Make thread-safe

type CalendarQueue[T any] struct {
	buckets    []*event[T]
	numBuckets int
	lastBucket int

	bucketWidth float32
	bucketTop   float32

	lastPrio float32

	size         int
	topThreshold int // TODO: rename
	botThreshold int // TODO rename
}

func (q *CalendarQueue[T]) Enqueue(entry event[T]) {
	i := int(entry.priority/q.bucketWidth) % q.numBuckets

	if q.buckets[i] == nil || q.buckets[i].priority > entry.priority {
		entry.next = q.buckets[i]
		q.buckets[i] = &entry
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
		current.next = &entry
	}

	q.size++
	if q.size > q.topThreshold {
		// TODO: resize
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
			if q.size < q.botThreshold {
				// resize bucket
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

	var minPrio float32
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
	q.bucketTop = (float32(n) + 1.5) * q.bucketWidth

	return q.Dequeue()
}

type event[T any] struct {
	data     T
	priority float32
	next     *event[T]
}

func NewEvent[T any](data T, priority float32) event[T] {
	return event[T]{
		data:     data,
		priority: priority,
		next:     nil,
	}
}
