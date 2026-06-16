import winston from "winston";

export const logger = winston.createLogger({
  transports: [
    new winston.transports.Console({
      level: "info",
      format: winston.format.combine(
        winston.format.colorize({
          message: true,
        }),
        // Likely a bug that 'info.message' has type 'unknown', but it's easy to workaround by converting to string
        // https://github.com/winstonjs/winston/issues/2528
        winston.format.printf((info) => String(info.message))
      ),
    }),
  ],
});
