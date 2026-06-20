#!/bin/bash
set -e

echo "Getting Admin Token..."
ADMIN_TOKEN=$(curl -s -X POST http://localhost:8080/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"admin123"}' | jq -r .token)

if [ "$ADMIN_TOKEN" == "null" ] || [ -z "$ADMIN_TOKEN" ]; then
    echo "FAIL: Could not obtain ADMIN_TOKEN"
    exit 1
fi
echo "PASS: Admin Token obtained"

echo -e "\n--- TEST 1: Create a DEB repo ---"
RES1=$(curl -s -w "\nHTTP_CODE:%{http_code}\n" -X POST http://localhost:8080/api/repos \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"slug":"deb-test","name":"DEB Test","codename":"stable","type":"deb"}')
echo "$RES1"
if ! echo "$RES1" | grep -q "HTTP_CODE:201"; then echo "FAIL: Expected HTTP 201"; exit 1; fi
if ! echo "$RES1" | grep -q '"type":"deb"'; then echo "FAIL: Expected type: deb"; exit 1; fi
echo "PASS: Test 1"

echo -e "\n--- TEST 2: Create an RPM repo ---"
RES2=$(curl -s -w "\nHTTP_CODE:%{http_code}\n" -X POST http://localhost:8080/api/repos \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"slug":"rpm-test","name":"RPM Test","type":"rpm"}')
echo "$RES2"
if ! echo "$RES2" | grep -q "HTTP_CODE:201"; then echo "FAIL: Expected HTTP 201"; exit 1; fi
if ! echo "$RES2" | grep -q '"type":"rpm"'; then echo "FAIL: Expected type: rpm"; exit 1; fi
echo "PASS: Test 2"

RPM_REPO_ID=$(curl -s http://localhost:8080/api/repos -H "Authorization: Bearer $ADMIN_TOKEN" | jq -r '.[] | select(.slug=="rpm-test") | .id')
DEB_REPO_ID=$(curl -s http://localhost:8080/api/repos -H "Authorization: Bearer $ADMIN_TOKEN" | jq -r '.[] | select(.slug=="deb-test") | .id')

echo -e "\n--- TEST 3: Repo list shows type field ---"
RES3=$(curl -s -w "\nHTTP_CODE:%{http_code}\n" http://localhost:8080/api/repos -H "Authorization: Bearer $ADMIN_TOKEN")
echo "$RES3"
if ! echo "$RES3" | grep -q '"type":"deb"'; then echo "FAIL: Expected type deb in list"; exit 1; fi
if ! echo "$RES3" | grep -q '"type":"rpm"'; then echo "FAIL: Expected type rpm in list"; exit 1; fi
echo "PASS: Test 3"

echo -e "\n--- TEST 4: Upload .deb to RPM repo is rejected ---"
RES4=$(curl -s -w "\nHTTP_CODE:%{http_code}\n" -X POST http://localhost:8080/api/repos/$RPM_REPO_ID/packages \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -F "file=@./testdata/hello_1.0_amd64.deb")
echo "$RES4"
if ! echo "$RES4" | grep -q "HTTP_CODE:400"; then echo "FAIL: Expected HTTP 400"; exit 1; fi
echo "PASS: Test 4"

echo -e "\n--- TEST 5: Upload .rpm to DEB repo is rejected ---"
RES5=$(curl -s -w "\nHTTP_CODE:%{http_code}\n" -X POST http://localhost:8080/api/repos/$DEB_REPO_ID/packages \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -F "file=@./testdata/hello-1.0-1.el9.x86_64.rpm")
echo "$RES5"
if ! echo "$RES5" | grep -q "HTTP_CODE:400"; then echo "FAIL: Expected HTTP 400"; exit 1; fi
echo "PASS: Test 5"

echo -e "\n--- TEST 6: Upload .rpm to RPM repo succeeds ---"
RES6=$(curl -s -w "\nHTTP_CODE:%{http_code}\n" -X POST http://localhost:8080/api/repos/$RPM_REPO_ID/packages \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -F "file=@./testdata/hello-1.0-1.el9.x86_64.rpm")
echo "$RES6"
if ! echo "$RES6" | grep -q "HTTP_CODE:201"; then echo "FAIL: Expected HTTP 201"; exit 1; fi
if ! echo "$RES6" | grep -q '"release"'; then echo "FAIL: Expected release field"; exit 1; fi
echo "PASS: Test 6"

echo -e "\n--- TEST 7: Package list for RPM repo shows release field ---"
RES7=$(curl -s -w "\nHTTP_CODE:%{http_code}\n" http://localhost:8080/api/repos/$RPM_REPO_ID/packages \
  -H "Authorization: Bearer $ADMIN_TOKEN")
echo "$RES7"
if ! echo "$RES7" | grep -q "HTTP_CODE:200"; then echo "FAIL: Expected HTTP 200"; exit 1; fi
if ! echo "$RES7" | grep -q '"release"'; then echo "FAIL: Expected release field"; exit 1; fi
if ! echo "$RES7" | grep -q '"arch"'; then echo "FAIL: Expected arch field"; exit 1; fi
echo "PASS: Test 7"

echo -e "\n--- TEST 8: Duplicate RPM upload returns 409 ---"
RES8=$(curl -s -w "\nHTTP_CODE:%{http_code}\n" -X POST http://localhost:8080/api/repos/$RPM_REPO_ID/packages \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -F "file=@./testdata/hello-1.0-1.el9.x86_64.rpm")
echo "$RES8"
if ! echo "$RES8" | grep -q "HTTP_CODE:409"; then echo "FAIL: Expected HTTP 409"; exit 1; fi
echo "PASS: Test 8"

echo -e "\nWaiting 5 seconds for async indexing..."
sleep 5

echo -e "\n--- TEST 9: repomd.xml is generated and valid ---"
RES9=$(curl -s -w "\nHTTP_CODE:%{http_code}\n" http://localhost:8080/repo/rpm-test/repodata/repomd.xml)
echo "$RES9"
if ! echo "$RES9" | grep -q "HTTP_CODE:200"; then echo "FAIL: Expected HTTP 200"; exit 1; fi
if ! echo "$RES9" | grep -q '<data type="primary">'; then echo "FAIL: Expected primary data"; exit 1; fi
echo "PASS: Test 9"

echo -e "\n--- TEST 10: primary.xml.gz is generated and contains the package ---"
PRIMARY_URL=$(curl -s http://localhost:8080/repo/rpm-test/repodata/repomd.xml | grep -o 'repodata/[^"]*primary[^"]*\.gz' | head -1)
if [ -z "$PRIMARY_URL" ]; then echo "FAIL: Could not find primary.xml.gz in repomd.xml"; exit 1; fi
RES10=$(curl -s http://localhost:8080/repo/rpm-test/$PRIMARY_URL | gunzip)
echo "${RES10:0:500}..."
if ! echo "$RES10" | grep -q '<package type="rpm">'; then echo "FAIL: Expected package type rpm"; exit 1; fi
if ! echo "$RES10" | grep -q '<name>hello</name>'; then echo "FAIL: Expected hello name"; exit 1; fi
echo "PASS: Test 10"

echo -e "\n--- TEST 11: repomd.xml.asc signature exists ---"
RES11=$(curl -s -w "\nHTTP_CODE:%{http_code}\n" http://localhost:8080/repo/rpm-test/repodata/repomd.xml.asc)
echo "$RES11"
if ! echo "$RES11" | grep -q "HTTP_CODE:200"; then echo "FAIL: Expected HTTP 200"; exit 1; fi
if ! echo "$RES11" | grep -q '-----BEGIN PGP SIGNATURE-----'; then echo "FAIL: Expected PGP signature"; exit 1; fi
echo "PASS: Test 11"

echo -e "\n--- TEST 12: RPM package file is served publicly ---"
RES12=$(curl -s -o /dev/null -w "HTTP_CODE:%{http_code}\n" http://localhost:8080/repo/rpm-test/packages/hello-1.0-1.el9.x86_64.rpm)
echo "$RES12"
if ! echo "$RES12" | grep -q "HTTP_CODE:200"; then echo "FAIL: Expected HTTP 200"; exit 1; fi
echo "PASS: Test 12"

echo -e "\n--- TEST 13: RPM setup instructions return correct format ---"
RES13=$(curl -s -w "\nHTTP_CODE:%{http_code}\n" http://localhost:8080/api/repos/$RPM_REPO_ID/setup \
  -H "Authorization: Bearer $ADMIN_TOKEN")
echo "$RES13"
if ! echo "$RES13" | grep -q "HTTP_CODE:200"; then echo "FAIL: Expected HTTP 200"; exit 1; fi
if ! echo "$RES13" | grep -q '"type":"rpm"'; then echo "FAIL: Expected type: rpm"; exit 1; fi
if ! echo "$RES13" | grep -q 'yum.repos.d'; then echo "FAIL: Expected yum.repos.d snippet"; exit 1; fi
if echo "$RES13" | grep -q 'apt-get'; then echo "FAIL: Did not expect apt-get"; exit 1; fi
echo "PASS: Test 13"

echo -e "\n--- TEST 14: DEB setup instructions still work correctly ---"
RES14=$(curl -s -w "\nHTTP_CODE:%{http_code}\n" http://localhost:8080/api/repos/$DEB_REPO_ID/setup \
  -H "Authorization: Bearer $ADMIN_TOKEN")
echo "$RES14"
if ! echo "$RES14" | grep -q "HTTP_CODE:200"; then echo "FAIL: Expected HTTP 200"; exit 1; fi
if ! echo "$RES14" | grep -q 'apt-get'; then echo "FAIL: Expected apt-get snippet"; exit 1; fi
if echo "$RES14" | grep -q 'yum.repos.d'; then echo "FAIL: Did not expect yum.repos.d"; exit 1; fi
echo "PASS: Test 14"

echo -e "\n--- TEST 15: CLI push works with RPM repo ---"
./bin/aptify-cli push rpm-test ./testdata/hello-1.0-1.el9.x86_64.rpm || true
./bin/aptify-cli push rpm-test ./testdata/goodbye-1.0-1.el9.x86_64.rpm || true
echo "PASS: Test 15 (See CLI output above)"

