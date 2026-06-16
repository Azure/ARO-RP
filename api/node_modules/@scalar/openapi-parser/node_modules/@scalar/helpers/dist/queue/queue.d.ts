/**
 * Represents a node in a singly linked list structure, used internally by the Queue.
 *
 * Example:
 *   const node = new Node<number>(42);
 *   console.log(node.data); // 42
 *   console.log(node.next); // null
 */
export declare class Node<T> {
    data: T;
    next: Node<T> | null;
    constructor(data: T);
}
/**
 * A generic queue implementation using a singly linked list.
 *
 * Example usage:
 *
 *   const q = new Queue<number>();
 *   q.enqueue(1);
 *   q.enqueue(2);
 *   q.enqueue(3);
 *   console.log(q.dequeue()); // 1
 *   console.log(q.peek());    // 2
 *   console.log(q.getSize()); // 2
 *   console.log(q.toString()); // "2 -> 3"
 *   console.log(q.isEmpty()); // false
 */
export declare class Queue<T> {
    front: Node<T> | null;
    rear: Node<T> | null;
    size: number;
    constructor();
    /**
     * Adds an element to the end of the queue.
     * @param data - The data to add to the queue.
     */
    enqueue(data: T): void;
    /**
     * Removes and returns the front element of the queue.
     * @returns The data from the removed node, or null if the queue is empty.
     */
    dequeue(): T | null;
    /**
     * Returns the front element of the queue without removing it.
     * @returns The front data, or null if the queue is empty.
     */
    peek(): T | null;
    /**
     * Checks whether the queue is empty.
     * @returns True if the queue has no elements, false otherwise.
     */
    isEmpty(): boolean;
    /**
     * Returns the number of elements in the queue.
     * @returns The size of the queue.
     */
    getSize(): number;
    /**
     * Returns a string representation of the queue.
     * @returns Elements of the queue separated by ' -> '.
     */
    toString(): string;
}
//# sourceMappingURL=queue.d.ts.map