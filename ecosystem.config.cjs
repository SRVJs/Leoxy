module.exports = {
  apps: [
    {
      name: "leoxy",
      script: "./leoxy",
      args: "run",
      instances: 2,
      exec_mode: "fork",
      increment_var: "PORT",
      env: {
        PORT: 4444
      }
    }
  ]
};
