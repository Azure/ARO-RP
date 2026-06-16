
import {convertJsonPath, normalizePath} from "../src/utils"
import { deepStrictEqual, strictEqual } from "assert";

describe("Test utils", () => {
  test("test resolve reference",async ()=>{
    strictEqual(normalizePath("C:\\test\\a.jaon"),"file:///C:/test/a.jaon")
  });

  test("test convert json path",async ()=>{
  deepStrictEqual(convertJsonPath({
      consumes: [
        "application/json"
      ]
    },["consumes","0"]),["consumes",0])
  });
});