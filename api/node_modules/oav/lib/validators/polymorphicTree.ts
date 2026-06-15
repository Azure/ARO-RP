// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.

/**
 * @class
 * Creates a tree by traversing the definitions where the parent model is the rootNode and child
 * model is one of it's children.
 */
export class PolymorphicTree {
  public name: string;
  public children: Map<string, PolymorphicTree>;
  /**
   * @constructor
   * Initializes a new instance of the PolymorphicTree
   *
   * @param {string} name- The name of the parent model
   * @param {Map<string, PolymorphicTree>} [children] - A map of zero or more children representing
   *    the child models in the inheritance chain
   */
  public constructor(name: string, children?: Map<string, PolymorphicTree>) {
    if (
      name === null ||
      name === undefined ||
      typeof name.valueOf() !== "string" ||
      name.trim().length === 0
    ) {
      throw new Error(
        "name is a required property of type string and it cannot be an empty string."
      );
    }

    if (children !== null && children !== undefined && !(children instanceof Map)) {
      throw new Error("children is an optional property of type Map<string, PolymorphicTree>.");
    }
    this.name = name;
    this.children = children || new Map();
  }

  /**
   * Adds a childObject to the PolymorphicTree. This method will not add the child again if it is
   * already present.
   *
   * @param {PolymorphicTree} childObj- A polymorphicTree representing the child model.
   * @returns {PolymorphicTree} childObj - The created child node.
   */
  public addChildByObject(childObj: PolymorphicTree): PolymorphicTree {
    if (childObj === null || childObj === undefined || !(childObj instanceof PolymorphicTree)) {
      throw new Error("childObj is a required parameter of type PolymorphicTree.");
    }

    if (!this.children.has(childObj.name)) {
      this.children.set(childObj.name, childObj);
    }
    return childObj;
  }
}
