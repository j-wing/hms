import sys
import urllib
import requests
import time

from selenium import webdriver
from selenium.webdriver.common.keys import Keys
from selenium.common.exceptions import NoSuchElementException
TEST_PAGE_URL = "http://bob.com/"
TEST_PAGE_PATH = "mypath"
TEST_PAGE_PATH_TWO = "mypath2"
TEST_ADD_CHAT_PATH = "mypath3"

BASE_URL = None
API_KEY = None 
fb_chat_id = 12345
API_ADD = ("POST", "/api/add")
API_REMOVE = ("DELETE", "/api/remove")
API_LIST = ("GET", "/api/list")
API_RESOLVE = ("GET", "/api/resolve")

STATUSES_SUCCESS = (200,)

def _get_form_fields(driver):
    path_field = driver.find_element_by_name("path")
    target_field = driver.find_element_by_name("target")

    return (path_field, target_field)

def _get_created_header(driver):
    return driver.find_element_by_class_name("bg-primary")

def _test_create_shortlink(driver, target_path, target_url):
    login(driver)
    path, target = _get_form_fields(driver)

    path.send_keys(target_path)
    target.send_keys(target_url)
    target.send_keys(Keys.RETURN)

    _test_header_link_goes_to(driver, target_url)

    return True

def _test_header_link_goes_to(driver, url):
    msg_header = _get_created_header(driver)
    link = msg_header.find_element_by_tag_name("a")
    link.click()
    assert driver.current_url == url
    return True


def _assert_api_status_code(endpoint, params, statuses_pass, statuses_fail, api_key=None, parse_json=True):
    method, path = endpoint

    if api_key is None:
        api_key = API_KEY
    full_params = {
            'apiKey': api_key
    }

    full_params.update(params)
    
    url = BASE_URL + path
    resp = getattr(requests, method.lower())(url, params=full_params)
    status = resp.status_code

    assert status not in statuses_fail, resp.text
    if statuses_pass:
        assert status in statuses_pass, resp.text

    if resp:
        if parse_json:
            return resp.json()
        return resp.text

def _assert_api_success_status(endpoint, params, *args, **kwargs):
    return _assert_api_status_code(endpoint, params, STATUSES_SUCCESS, [], *args, **kwargs)

def _assert_api_failure_status(endpoint, params, *args, **kwargs):
    return _assert_api_status_code(endpoint, params, [], STATUSES_SUCCESS, *args, **kwargs)

def test_can_create_random_path(driver):
   return _test_create_shortlink(driver, "", TEST_PAGE_URL) 

def login(driver, email="test@example.com", admin=True):
    if driver.current_url.count("/_ah/login") != 0:
        driver.find_element_by_id("email").send_keys((Keys.BACKSPACE*20) + email)
        if admin:
            driver.find_element_by_id("admin").click()
        driver.find_element_by_id("submit-login").click()

def test_can_create_explicit_path(driver):
    return _test_create_shortlink(driver, TEST_PAGE_PATH, TEST_PAGE_URL)


def test_access_requires_login(driver):
    assert driver.current_url.count("/_ah/login") != 0
    driver.get("%s/add" % BASE_URL)
    assert driver.current_url.count("/_ah/login") != 0
    login(driver)
    return True

def test_unauthorized_email_fails(driver):
    login(driver, email="thisshouldnot@work.com")
    try:
        driver.find_element_by_name("path")
    except NoSuchElementException:
        driver.delete_all_cookies()
        return True
    driver.delete_all_cookies()
    return False

def test_can_use_quick_add(driver):
    driver.get("{0}add?target={1}".format(driver.current_url, TEST_PAGE_URL))
    return _test_header_link_goes_to(driver, TEST_PAGE_URL)

def test_multiple_links_with_same_path_fail(driver):
    try:
        _test_create_shortlink(driver, TEST_PAGE_PATH, TEST_PAGE_URL)
    except NoSuchElementException:
        return True
    return False

def test_cant_add_api_key_if_not_logged_in(driver):
    driver.get("%s/add_api_key?owner=test@example.com" % BASE_URL)
    assert driver.current_url.count("/_ah/login") != 0
    return True

def test_can_add_api_key(driver):
    global API_KEY
    driver.get("%s/add_api_key?owner=test@example.com" % BASE_URL)
    assert "error" not in driver.page_source
    API_KEY = driver.page_source
    return True

def test_no_apikey_fails(driver):
    _assert_api_failure_status(API_RESOLVE, api_key="")
    return True

def test_can_add_chat(driver):
    driver.get("%s/add_chat?name=HMS&fbID=%s" % (BASE_URL, fb_chat_id))
    assert "error" not in driver.page_source
    return True
    
def test_api_resolve(driver):
    _assert_api_failure_status(API_RESOLVE, {})
    result = _assert_api_success_status(API_RESOLVE, {'path': TEST_PAGE_PATH})
    assert result['Success'], result
    assert result['Result']['TargetURL'] == TEST_PAGE_URL, result

    result = _assert_api_success_status(API_RESOLVE, {
        'path': TEST_PAGE_PATH,
        'chatID': fb_chat_id
    })

    assert not result['Success'], result

    result = _assert_api_success_status(API_RESOLVE, {
        'path': TEST_ADD_CHAT_PATH,
        'chatID': fb_chat_id
    })
    assert result['Success'], result
    return True

def test_api_remove(driver):
    #_assert_api_failure_status(API_REMOVE, {})

    result = _assert_api_success_status(API_REMOVE, {
        'path': TEST_PAGE_PATH,
        'chatID': fb_chat_id
    })
    assert result['NumRemoved'] == 0

    result = _assert_api_success_status(API_REMOVE, {'path': TEST_PAGE_PATH})
    assert result['Success']
    assert result['NumRemoved'] == 1

    time.sleep(.5)
    result = _assert_api_success_status(API_RESOLVE, {'path': TEST_PAGE_PATH})
    assert not result['Success']

    result = _assert_api_success_status(API_REMOVE, {'path': TEST_PAGE_PATH})
    assert result['Success']
    assert result['NumRemoved'] == 0

    result = _assert_api_success_status(API_REMOVE, {'path': TEST_ADD_CHAT_PATH})
    assert result['NumRemoved'] == 0

    result = _assert_api_success_status(API_REMOVE, {
        'path': TEST_ADD_CHAT_PATH,
        'chatID': fb_chat_id
    })
    assert result['Success']
    assert result['NumRemoved'] == 1, result

    return True

def test_api_add(driver):
    _assert_api_failure_status(API_ADD, {})
    
    result = _assert_api_success_status(API_ADD, {
        'target': TEST_PAGE_URL,
        "creator": "Test Tester"
    })

    assert result['Success']

    time.sleep(.5)
    driver.get(result['ResultURL'])
    assert driver.current_url == TEST_PAGE_URL,driver.current_url 

    result = _assert_api_success_status(API_ADD, {
        'target': TEST_PAGE_URL,
        'creator': 'Test Tester',
        'chatID': fb_chat_id,
        'path': TEST_ADD_CHAT_PATH,
    })

    assert result['Success']
    time.sleep(.5)
    driver.get(result['ResultURL'])
    assert driver.current_url == TEST_PAGE_URL,driver.current_url 

    return True

TESTS = [
        #test_unauthorized_email_fails,
        test_cant_add_api_key_if_not_logged_in,
        test_access_requires_login,
        test_can_create_explicit_path,
        test_can_create_random_path,
        test_multiple_links_with_same_path_fail,
        test_can_add_api_key,
        test_can_add_chat,
        test_api_add,
        test_api_resolve,
        test_api_remove,
    ]

def main(url, tests):
    global BASE_URL
    BASE_URL = url
    driver = webdriver.Firefox()
    
    if tests is None:
        tests = range(len(TESTS))

    failed = []
    i = 0
    for test in TESTS:
        if i not in tests:
            i += 1
            continue
        driver.get(url)
        print "Running", test.func_name
        try:
            result = test(driver)
        except Exception:
            print "Exception encountered running test", test.func_name
            raise

        if not result:
            print "Test failed!"
            failed.append(test)
        i += 1

    failed_count = len(failed)
    passed = len(tests) - failed_count
    print "Results:", passed, "out of", len(tests), "passed (", failed_count, "failed). "
    print "\n\t - ".join(failed_test.func_name for failed_test in failed)
    driver.quit()

if __name__ == "__main__":
    if len(sys.argv) > 1:
        tests = [int(i) for i in sys.argv[1].split(",")]
    else:
        tests = None
    url = "http://localhost:8080"

    main(url, tests=tests)
