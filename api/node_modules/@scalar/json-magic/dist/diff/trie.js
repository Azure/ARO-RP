class TrieNode {
  constructor(value, children) {
    this.value = value;
    this.children = children;
  }
}
class Trie {
  root;
  constructor() {
    this.root = new TrieNode(null, {});
  }
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
  addPath(path, value) {
    let current = this.root;
    for (const dir of path) {
      if (current.children[dir]) {
        current = current.children[dir];
      } else {
        current.children[dir] = new TrieNode(null, {});
        current = current.children[dir];
      }
    }
    current.value = value;
  }
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
  findMatch(path, callback) {
    let current = this.root;
    for (const dir of path) {
      if (current.value !== null) {
        callback(current.value);
      }
      const next = current.children[dir];
      if (!next) {
        return;
      }
      current = next;
    }
    const dfs = (current2) => {
      for (const child of Object.keys(current2?.children ?? {})) {
        if (current2 && Object.hasOwn(current2.children, child)) {
          dfs(current2?.children[child]);
        }
      }
      if (current2?.value) {
        callback(current2.value);
      }
    };
    dfs(current);
  }
}
export {
  Trie,
  TrieNode
};
//# sourceMappingURL=trie.js.map
