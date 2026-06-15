function queueTask(queue, task) {
  return {
    ...queue,
    tasks: [...queue.tasks, task]
  };
}
export {
  queueTask
};
//# sourceMappingURL=queueTask.js.map
