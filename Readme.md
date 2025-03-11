

# link extraction
1. Get the HTML content of the webpage using the requests library.
2. Parse the HTML content using the BeautifulSoup library.
3. Find all the dropdown options in the select tag.
4. Iterate through each option and decode the base64 encoded value.
5. Split the decoded value into lines and find the line containing the iframe src.
6. Extract the src attribute from the iframe tag and store it in a dictionary.
    ## Rumble
    - Iterate through the dictionary and find the key containing the word "rumble".
    - Get the HTML content of the iframe src using the requests library.
    - Clean the HTML content by removing all the backslashes.
    - Find all the mp4 urls in the cleaned HTML content using the re library.
    - Sort the urls in descending order based on the quality.
    - Print all the urls.
    ## other
