/**
 * Trie data structure
 *
 * Read more: https://en.wikipedia.org/wiki/Trie
 */
/**
 * Represents a node in the trie data structure.
 * Each node can store a value and has a map of child nodes.
 *
 * @template Value - The type of value that can be stored in the node
 */
export declare class TrieNode<Value> {
    value: Value | null;
    children: Record<string, TrieNode<Value>>;
    constructor(value: Value | null, children: Record<string, TrieNode<Value>>);
}
/**
 * A trie (prefix tree) data structure implementation.
 * This class provides efficient storage and retrieval of values associated with string paths.
 *
 * @template Value - The type of value to store at each node
 *
 * @example
 * const trie = new Trie<number>()
 * trie.addPath(['a', 'b', 'c'], 1)
 * trie.addPath(['a', 'b', 'd'], 2)
 * trie.findMatch(['a', 'b'], (value) => console.log(value)) // Logs: 1, 2
 */
export declare class Trie<Value> {
    private root;
    constructor();
    /**
     * Adds a value to the trie at the specified path.
     * Creates new nodes as needed to build the path.
     *
     * @param path - Array of strings representing the path to store the value
     * @param value - The value to store at the end of the path
     *
     * @example
     * const trie = new Trie<number>()
     * trie.addPath(['users', 'john', 'age'], 30)
     */
    addPath(path: string[], value: Value): void;
    /**
     * Finds all matches along a given path in the trie.
     * This method traverses both the exact path and all deeper paths,
     * executing a callback for each matching value found.
     *
     * The search is performed in two phases:
     * 1. Traverse the exact path, checking for matches at each node
     * 2. Perform a depth-first search from the end of the path to find all deeper matches
     *
     * @param path - Array of strings representing the path to search
     * @param callback - Function to execute for each matching value found
     *
     * @example
     * const trie = new Trie<number>()
     * trie.addPath(['a', 'b', 'c'], 1)
     * trie.addPath(['a', 'b', 'd'], 2)
     * trie.findMatch(['a', 'b'], (value) => console.log(value)) // Logs: 1, 2
     */
    findMatch(path: string[], callback: (value: Value) => void): void;
}
//# sourceMappingURL=trie.d.ts.map