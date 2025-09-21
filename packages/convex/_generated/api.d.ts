/* eslint-disable */
/**
 * Generated `api` utility.
 *
 * THIS CODE IS AUTOMATICALLY GENERATED.
 *
 * To regenerate, run `npx convex dev`.
 * @module
 */

import type {
  ApiFromModules,
  FilterApi,
  FunctionReference,
} from "convex/server";
import type * as files from "../files.js";
import type * as filesystem from "../filesystem.js";
import type * as functions from "../functions.js";
import type * as model_directories from "../model/directories.js";
import type * as model_error from "../model/error.js";
import type * as model_files from "../model/files.js";
import type * as model_filesystem from "../model/filesystem.js";
import type * as model_user from "../model/user.js";
import type * as users from "../users.js";

/**
 * A utility for referencing Convex functions in your app's API.
 *
 * Usage:
 * ```js
 * const myFunctionReference = api.myModule.myFunction;
 * ```
 */
declare const fullApi: ApiFromModules<{
  files: typeof files;
  filesystem: typeof filesystem;
  functions: typeof functions;
  "model/directories": typeof model_directories;
  "model/error": typeof model_error;
  "model/files": typeof model_files;
  "model/filesystem": typeof model_filesystem;
  "model/user": typeof model_user;
  users: typeof users;
}>;
export declare const api: FilterApi<
  typeof fullApi,
  FunctionReference<any, "public">
>;
export declare const internal: FilterApi<
  typeof fullApi,
  FunctionReference<any, "internal">
>;
