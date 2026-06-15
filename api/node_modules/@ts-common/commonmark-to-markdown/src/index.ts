import * as cm from "commonmark"
import * as it from "@ts-common/iterator"
var fm = require('front-matter')

export interface MarkDownEx {
  readonly frontMatter?: string
  readonly markDown: cm.Node
}

export type NodeType = cm.NodeType

export const createNode = (type: NodeType, ...children: readonly cm.Node[]) => {
  const result = new cm.Node(type)
  for (const c of children) {
    result.appendChild(c)
  }
  return result
}

export const createText = (literal: string) => {
  const result = createNode("text")
  result.literal = literal
  return result
}

export const createCodeBlock = (info: string, literal: string) => {
  const result = createNode("code_block")
  result.info = info
  result.literal = literal
  return result
}

export const iterate = (node: cm.Node) => it.iterable(function *() {
  let c = node.firstChild
  while (c !== null) {
    yield c
    c = c.next
  }
})

export const parse = (fileContent: string): MarkDownEx => {
  const result = fm(fileContent)
  const parser = new cm.Parser()
  return {
    frontMatter: result.frontmatter,
    markDown: parser.parse(result.body)
  }
}

export const markDownExToString = (mde: MarkDownEx): string => {
  const md = unescape(commonmarkToString(mde.markDown))
  return mde.frontMatter === undefined ? md : `---\n${mde.frontMatter}\n---\n${md}`
}

const commonmarkToString = (root: cm.Node) => {
  let walker = root.walker();
  let event;
  let output = "";
  while ((event = walker.next())) {
    let curNode = event.node;

    const leaving = render.leaving[curNode.type]
    if (!event.entering && leaving !== undefined) {
      output += leaving(curNode, event.entering);
    }
    const entering = render.entering[curNode.type]
    if (event.entering && entering !== undefined) {
      output += entering(curNode, event.entering);
    }
  }

  output = output.replace(/\n$/, "");

  return output;
}

type Func = (node: cm.Node, b: unknown) => unknown

interface Render {
  readonly entering: {
    readonly [key in NodeType]?: Func
  }
  readonly leaving: {
    readonly [key in NodeType]?: Func
  }
}

const indent = (node: cm.Node|null): string =>
  node !== null ? indent(node.parent) + (node.type === "item" ? "  " : "") : ""

const render : Render = {
  entering: {
    text: (node: cm.Node) => node.literal,
    softbreak: () => "\n",
    linebreak: () => "\n",
    emph: () => "*",
    strong: () => "**",
    html_inline: () => "`",
    link: () => "[",
    image: () => {},
    code: (node: cm.Node) => `\`${node.literal}\``,
    document: () => "",
    paragraph: () => "",
    block_quote: () => "> ",
    item: (node: cm.Node) =>
      `${indent(node.parent)}${{ bullet: "*", ordered: `1${node.listDelimiter}` }[node.listType]} `,
    list: () => "",
    heading: (node: cm.Node) =>
      Array(node.level)
        .fill("#")
        .join("") + " ",
    code_block: (node: cm.Node) =>
      `\`\`\` ${node.info}\n${node.literal}\`\`\`\n\n`,
    html_block: (node: cm.Node) => node.literal,
    thematic_break: () => "---\n\n",
    custom_inline: () => {},
    custom_block: () => {},
  },

  leaving: {
    heading: () => "\n\n",
    paragraph: () => "\n\n",
    link: (node: cm.Node) => `](${node.destination})`,
    strong: () => "**",
    emph: () => "*",
  }
};
