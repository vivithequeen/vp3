import React from "react";
import { render, Box, Text, Spinner, useInput, useTui, ScrollView } from "@orchetron/storm";

function App() {
    const { exit } = useTui();
    useInput((e) => { if (e.key === "c" && e.ctrl) exit(); });

    return (
        <Box borderStyle="round" borderColor="#F4A4BF" padding={1}>
            <Text>Title                         Artist                   Album</Text>
            <ScrollView flex={0}>
                    <Text>Curses - The Crane Wives</Text>


            </ScrollView>
        </Box>
        

    );
}

render(<App />).waitUntilExit();