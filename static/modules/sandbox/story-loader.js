/**
 * Loads a story module and its default arguments.
 * @param {string} storyModulePath - The path to the story module.
 * @param {string} storyKey - The key of the story to load.
 * @returns {Promise<{storyDefaultArgs: object, module: object}>} An object containing the story's default args and the loaded module.
 * @throws {Error} If the module or story cannot be loaded, or if args are invalid.
 */
export async function loadStory(storyModulePath, storyKey) {
  console.log(
    `[StoryLoader] Attempting to load story: ${storyKey} from module: ${storyModulePath}`
  );
  try {
    const module = await import(storyModulePath);
    if (
      module &&
      module[storyKey] &&
      typeof module[storyKey].args === "object"
    ) {
      console.log(
        `[StoryLoader] Successfully loaded module and found story '${storyKey}'.`
      );
      const storyDefaultArgs = JSON.parse(
        JSON.stringify(module[storyKey].args)
      );
      return { storyDefaultArgs, module }; // Returning module for now, might be refined later
    } else {
      let errorDetail = `Story '${storyKey}' or its .args object not found/valid in module.`;
      if (!module) {
        errorDetail = `Module ${storyModulePath} not loaded.`;
      } else if (!module[storyKey]) {
        errorDetail = `Story key '${storyKey}' not found in module. Available exports: ${Object.keys(
          module
        ).join(", ")}`;
      } else if (typeof module[storyKey].args !== "object") {
        errorDetail = `Story '${storyKey}' found, but its .args is not an object: ${typeof module[
          storyKey
        ].args}`;
      }
      console.error(`[StoryLoader] Error: ${errorDetail}`);
      throw new Error(errorDetail);
    }
  } catch (err) {
    console.error(
      `[StoryLoader] Failed to import story module ${storyModulePath}:`,
      err
    );
    throw new Error(
      `Failed to load story module ${storyModulePath}. Original error: ${err.message}`
    );
  }
}
