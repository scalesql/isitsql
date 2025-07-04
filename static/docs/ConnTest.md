ConnTest.exe
============
`conntest.exe` is simple utilty to test SQL Server connections using the same code as `IsItSQL`.
It only supports trusted connections.  

Usage
-----
Running with no parameters just lists the best ODBC drivers.  The parameters are:

```
Usage of conntest.exe:
  -driver string
        driver to use
  -fqdn string
        fqdn to test connecting
```

Sample Output
------
```
conntest.exe -fqdn D40\SQL2019 -driver "ODBC Driver 18 for SQL Server" 

2023/06/25 08:53:34 found driver: ODBC Driver 11 for SQL Server
2023/06/25 08:53:34 found driver: ODBC Driver 13 for SQL Server
2023/06/25 08:53:34 found driver: ODBC Driver 17 for SQL Server
2023/06/25 08:53:34 found driver: ODBC Driver 18 for SQL Server
2023/06/25 08:53:34 found driver: SQL Server
2023/06/25 08:53:34 found driver: SQL Server Native Client 11.0
2023/06/25 08:53:34 ---------------------------------------------------------
2023/06/25 08:53:34 best driver: ODBC Driver 18 for SQL Server
2023/06/25 08:53:34 ---------------------------------------------------------
2023/06/25 08:53:34 fqdn: D40\SQL2019
2023/06/25 08:53:34 driver: ODBC Driver 18 for SQL Server
2023/06/25 08:53:34 connection string: Driver={ODBC Driver 18 for SQL Server}; Server=D40\SQL2019; Trusted_Connection=Yes; App=conntest.exe; Encrypt=Optional;
2023/06/25 08:53:34 @@SERVERNAME: D40\SQL2019
2023/06/25 08:53:34 auth: NTLM  interface: ODBC  version: 7
```
