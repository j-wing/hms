import sys

from selenium import webdriver
from selenium.webdriver.common.keys import Keys
from selenium.common.exceptions import NoSuchElementException
TEST_PAGE_URL = "http://bob.com/"
TEST_PAGE_PATH = "mypath"
TEST_PAGE_PATH_TWO = "mypath2"

BASE_URL = None


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

def test_can_create_random_path(driver):
   return _test_create_shortlink(driver, "", TEST_PAGE_URL) 

def login(driver, email="test@example.com"):
    if driver.current_url.count("/_ah/login") != 0:
        driver.find_element_by_id("email").send_keys((Keys.BACKSPACE*20) + email)
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

TESTS = [
        test_unauthorized_email_fails,
        test_access_requires_login,
        test_can_create_explicit_path,
        test_can_create_random_path,
#        test_can_use_quick_add,
        test_multiple_links_with_same_path_fail,
        ]

def main(url):
    global BASE_URL
    BASE_URL = url
    driver = webdriver.Firefox()

    failed = []
    for test in TESTS:
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

    failed_count = len(failed)
    passed = len(TESTS) - failed_count
    print "Results:", passed, "out of", len(TESTS), "passed (", failed_count, "failed). "
    print "\n\t - ".join(failed_test.func_name for failed_test in failed)
    driver.quit()

if __name__ == "__main__":
    if len(sys.argv) > 2:
        url = sys.argv[-1]
    else:
        url = "http://localhost:8080"

    main(url)
