const localDateTimeFormatter = new Intl.DateTimeFormat('en-CA', {
  calendar: 'gregory',
  numberingSystem: 'latn',
  year: 'numeric',
  month: '2-digit',
  day: '2-digit',
  hour: '2-digit',
  minute: '2-digit',
  second: '2-digit',
  fractionalSecondDigits: 3,
  hourCycle: 'h23',
  timeZoneName: 'longOffset',
});

export function localTimestamp(date) {
  const parts = Object.fromEntries(
    localDateTimeFormatter.formatToParts(date).map(({ type, value }) => [type, value]),
  );
  const offset = parts.timeZoneName === 'GMT' ? '+00:00' : parts.timeZoneName.replace(/^GMT/, '');

  return `${parts.year}-${parts.month}-${parts.day}`
    + `T${parts.hour}:${parts.minute}:${parts.second}`
    + `.${parts.fractionalSecond}${offset}`;
}

export function localNow() {
  return localTimestamp(new Date());
}
