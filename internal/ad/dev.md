* Demo in private Azure domain
* DEV on box02 using bguser@demo.loc
    * It has VSCode, etc.
    * Password is in the Vault
* Don't forget to enable my local IP in Azure restrictions for RDP

Groups
------
* CN=sql-sysadmins,CN=Users,DC=demo,DC=loc
* CN=sql backup shares,OU=Security Groups,OU=Groups,OU=Org,DC=domain,DC=com
* CN=eus-sql-dpa-sysadmin,OU=org-sqlgroups,OU=org,DC=core,DC=org,DC=us,DC=loc

- if POST
  - do all the login stuff
  - if OK
    - write the cookie
    - redirect to "/login"
  - else RenderFS with error 
- if GET
  - are we logged in? (read the cookie)
  - display login/logout buttons
  - renderFS

/*

    Common error messages

    Bad password:
    Error: testLogin2: Bind: LDAP Result Code 49 "Invalid Credentials": 80090308: LdapErr: DSID-0C0903D0, comment: AcceptSecurityContext error, data 52e, v2580

    With invalid domain:
    testLogin2: getLdapConnection: Dial: LDAP Result Code 200 "Newwork Error": dial tcp: lookup nodomain.here.local: no such host

    User = asdfasfasdf (no domain)
    Error: parsename: parseName: failed


*/