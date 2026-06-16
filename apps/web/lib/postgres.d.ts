declare module "postgres" {
  type Sql = <T = Record<string, unknown>>(
    strings: TemplateStringsArray,
    ...values: unknown[]
  ) => Promise<T[]>;

  export default function postgres(
    connectionString: string,
    options?: Record<string, unknown>,
  ): Sql;
}
